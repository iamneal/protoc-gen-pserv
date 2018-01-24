package main

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	gp "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

const PKG_PREFIX = "pserv-"

func generate(filepaths []string, protos []*gp.FileDescriptorProto) ([]*File, error, error) {
	oracle := Oracle{protos}
	out := make([]*File, 0)
	pkgs := oracle.Packages()
	for _, pkg := range pkgs {
		files := oracle.FilesIn(&pkg)
		out = append(out, &File{
			Name:    pkg.Name + "/service.generated.proto",
			Content: "",
		})
		lo := out[len(out)-1]
		lo.P(`syntax = "proto3";`, "\n", "package ", pkg.Name, ";\n")
		if pkg.GoPkg != "" {
			lo.P(`option go_package="`, pkg.GoPkg, `";`, "\n")
		}
		lo.P(`import "persist/options.proto";`, "\n")
		lo.P("service ", "Gen", strings.Replace(pkg.Name, ".", "_", -1), "{\n")
		lo.P("\toption (persist.service_type) = SPANNER;\n")

		for _, f := range files {
			if oracle.IsDependency(f.GetName()) {
				continue
			}
			srcCode, err := ioutil.ReadFile(path.Clean(fmt.Sprintf("./%s", f.GetName())))
			if err != nil {
				return nil, nil, err
			}
			srcLines := strings.Split(string(srcCode), "\n")

			// lookup all this files comments
			for _, l := range f.SourceCodeInfo.Location {
				// grab whole messages with a leading comment
				if l.LeadingComments != nil &&
					len(l.Path) == 2 && l.Path[0] == 4 &&
					(l.Path[1] == 0 || l.Path[1] == 1) {
					// must contain our subtext
					if strings.HasPrefix(strings.Trim(*l.LeadingComments, " \t"), PKG_PREFIX) {
						var important string
						if len(l.Span) == 3 {
							important = srcLines[l.Span[0]]
						} else if len(l.Span) == 4 {
							for i := l.Span[0]; i < l.Span[2]; i++ {
								important += srcLines[i] + "\n"
							}
						}
						// find the message by parsing the message name
						// concatting to the package
						// and locating in files in our package
						_, msg := oracle.GetDescriptorForComment(f, important)
						oracle.WriteCrud(lo, msg, l.GetLeadingComments())

					}
				}
			}
		}
		lo.P("}\n")
	}

	return out, nil, nil
}

type Oracle struct {
	protos []*gp.FileDescriptorProto
}

// bad function name.  comment is not a comment, but a source code snippet
func (o Oracle) GetDescriptorForComment(file *gp.FileDescriptorProto, comment string) (string, *gp.DescriptorProto) {
	// find message location in this comment
	msgIdx := strings.Index(comment, "message")
	if msgIdx < 0 {
		return "", nil
	}
	//after message, find the first 'bracket'
	bracketIdx := strings.Index(comment[msgIdx:], "{")
	// trim the whitespace between the message and the bracket
	name := strings.Trim(comment[msgIdx+len("message")+1:bracketIdx], " \n\t\r")
	pkg := &Package{Name: file.GetPackage()}
	messages := o.MessagesIn(pkg)
	for _, msg := range messages {
		if msg.GetName() == name ||
			msg.GetName() == pkg.Name+"."+name ||
			msg.GetName() == "."+pkg.Name+"."+name {
			return name, msg
		}
	}
	return name, nil
}

func (o Oracle) IsDependency(name string) bool {
	for _, f := range o.protos {
		for _, d := range f.Dependency {
			if name == d {
				return true
			}
		}
	}
	return false
}
func (o Oracle) Packages() []Package {
	pkgs := make(map[Package]struct{})

	for _, f := range o.protos {
		pkgs[Package{
			Name: f.GetPackage(),
			GoPkg: func() string {
				if opts := f.GetOptions(); opts != nil {
					return opts.GetGoPackage()
				}
				return ""
			}(),
		}] = struct{}{}
	}
	out := make([]Package, 0)

	for p, _ := range pkgs {
		out = append(out, p)
	}
	return out
}

func (o Oracle) FilesIn(p *Package) []*gp.FileDescriptorProto {
	var out []*gp.FileDescriptorProto

	for _, f := range o.protos {
		if f.GetPackage() == p.Name {
			out = append(out, f)
		}
	}
	return out
}

func (o Oracle) MessagesIn(p *Package) []*gp.DescriptorProto {
	descs := make(map[string]*gp.DescriptorProto)

	for _, f := range o.protos {
		if o.IsDependency(f.GetName()) {
			continue
		}
		if f.GetPackage() != p.Name {
			continue
		}
		for _, m := range f.GetMessageType() {
			descs[m.GetName()] = m
		}

	}
	out := make([]*gp.DescriptorProto, 0)

	for _, d := range descs {
		out = append(out, d)
	}
	return out
}

func (o Oracle) WriteCrud(f *File, msg *gp.DescriptorProto, comment string) {
	lines := strings.Split(comment, "\n")
	table := func() string {
		for _, l := range lines {
			if strings.HasPrefix(l, PKG_PREFIX+"table=") {
				return strings.Trim(l[len(PKG_PREFIX+"table="):], "\n\t \r")
			}
		}
		return ""
	}()
	pk := func() []string {
		for _, l := range lines {
			if strings.HasPrefix(l, PKG_PREFIX+"pk=") {
				pks := strings.Split(l[len(PKG_PREFIX+"pk="):], ",")
				for i, p := range pks {
					pks[i] = strings.Trim(p, "\n\t \r")
				}
				return pks
			}
		}
		return nil
	}()
	if table == "" || pk == nil {
		return
	}
	inPk := func(s string) bool {
		for _, p := range pk {
			if s == p {
				return true
			}
		}
		return false
	}
	notPks := func() (out []string) {
		for _, f := range msg.GetField() {
			if !inPk(f.GetName()) {
				out = append(out, f.GetName())
			}
		}
		return
	}()
	all := append(pk, notPks...)
	n := msg.GetName()
	f.P("\trpc Insert", n, "s(stream ", n, ") returns (", n, "){\n")
	f.P("\t\toption (persist.ql) = {\n\t\t\tquery:[")
	f.P(`"INSERT INTO `, table, " (")
	for i := 0; i < len(all)-1; i++ {
		f.P(all[i], ",")
	}
	f.P(all[len(all)-1], ") VALUES (")
	for i := 0; i < len(all)-1; i++ {
		f.P("@", all[i], ",")
	}
	f.P(all[len(all)-1], `)"`)
	f.P("],\n\t\t};\n")
	f.P("\t};\n")

	f.P("\trpc Select", n, "ByPk(", n, ") returns(", n, "){\n")
	f.P("\t\toption (persist.ql) = {\n\t\t\tquery:[")
	f.P(`"SELECT `)
	for i := 0; i < len(all)-1; i++ {
		f.P(all[i], ",")
	}
	f.P(all[len(all)-1], ") FROM ", table, "WHERE ")
	for i := 0; i < len(pk)-1; i++ {
		f.P(pk[i], "=@", pk[i], " && ")
	}
	f.P(pk[len(pk)-1], "=@", pk[len(pk)-1], `"`)
	f.P("],\n\t\t};\n")
	f.P("\t};\n")

	f.P("\trpc Delete", n, "(", n, ") returns(", n, "){\n")
	f.P("\t\toption (persist.ql) = {\n\t\t\tquery:[")
	f.P(`"DELETE FROM `, table, " VALUES(")
	for i := 0; i < len(pk)-1; i++ {
		f.P("@", pk[i], ",")
	}
	f.P("@", pk[len(pk)-1], `)"`)
	f.P("],\n\t\t};\n")
	f.P("\t};\n")

	f.P("\trpc Update", n, "(", n, ") returns(", n, "){\n")
	f.P("\t\toption (persist.ql) = {\n\t\t\tquery:[")
	f.P(`"UPDATE `, table, " ")
	for i := 0; i < len(notPks)-1; i++ {
		f.P("set ", notPks[i], "=@", notPks[i], ", ")
	}
	f.P("set ", notPks[len(notPks)-1], "=@", notPks[len(notPks)-1], " ")
	f.P("PK(")
	for i := 0; i < len(pk)-1; i++ {
		f.P(pk[i], "=@", pk[i], ",")
	}
	f.P(pk[len(pk)-1], "=@", pk[len(pk)-1], `)"`)
	f.P("],\n\t\t};\n")
	f.P("\t};\n")
}

type File struct {
	Name    string
	Content string
}

func (f *File) P(args ...interface{}) {
	for _, a := range args {
		f.Content += fmt.Sprintf("%s", a)
	}
}

type Package struct {
	Name  string // name of the protobuf package given to the descriptor
	GoPkg string // the go_package option if there is one
}

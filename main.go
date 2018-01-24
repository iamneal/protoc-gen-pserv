package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
)

func main() {
	var req plugin_go.CodeGeneratorRequest

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(fmt.Errorf("got error reading from stdin: %v", err))
	}
	if err := proto.Unmarshal(data, &req); err != nil {
		panic(fmt.Errorf("got error unmarshaling request: %v", err))
	}
	files, err, internalErr := generate(req.GetFileToGenerate(), req.GetProtoFile())
	if internalErr != nil {
		panic(fmt.Errorf("error generating: %v", err))
	}
	resp := plugin_go.CodeGeneratorResponse{
		Error: func() (out *string) {
			if err != nil {
				*out = fmt.Sprintf("%s", err)
				return
			}
			return
		}(),
		File: func() (out []*plugin_go.CodeGeneratorResponse_File) {
			if err != nil {
				return
			}
			for _, f := range files {
				out = append(out, &plugin_go.CodeGeneratorResponse_File{
					Name:    &f.Name,
					Content: &f.Content,
				})
			}
			return
		}(),
	}
	data, err = proto.Marshal(&resp)
	if err != nil {
		panic(fmt.Errorf("error marshaling the response: %v", err))
	}
	if _, err := os.Stdout.Write(data); err != nil {
		panic(fmt.Errorf("error writing to std out: %v", err))
	}
}

# protoc-gen-pserv

This project is just a helper plugin to generate a very basic crud service by looking at comments on a protobuf message.

It looks at all the .proto files in a package, groups them together, and writes into that package a service with basic crud
operations performed with persist.  It is not exhastive, it definitely has bugs, and was made to solve a very specific pain point.

Feel free to use the plugin,  but it probably wont get a ton more features.


This whole repo is small enough to serve as an example project.

The make file points to the protoc command used to generate the service file.

```make
protos:
	protoc -I. -I$$GOPATH/src -I./test \
		--plugin=./protoc-gen-pserv \
		--pserv_out=. ./test/*.proto
	# protoc -I. -I$$GOPATH/src -I./test \
	# 	--persist_out=persist_root=github.com/iamneal/protoc-gen-pserv/test:. --go_out=plugins=grpc:. \
	# 	./test/*.proto
```
the commented out protoc command is what would be run after.

notice it is not all squished into one command.  It cannot be that way, because it is writing a proto file that will
be passed to persist.  This plugin needs to be run seperately __before__ any other protoc commands



also notice the persist_root  option set in the second protoc command.  This plugin was built hastily, and does not yet respect the go_package option.  So make sure you are using correct package names,  and be aware that if you do not,  you might get your file generated into the wrong place.


the test directory has user.proto, and additional.proto
```proto
# user.proto
syntax = "proto3";

package test;

//pserv-table=users
//pserv-pk=id, name
message User {
    //test stuff in the comment    
	int64 id = 1;
	string name = 2; //test extra at the end of things
	Friends friends = 3;
}

//pserv-table=friends
//pserv-pk=logger
message Friends {
    string names  = 1;
    
    string logger = 2;
}

# additional.proto
syntax="proto3";
package test;

//pserv-table=extra
//pserv-pk=f1,f3,f4
message Extra {
    string f1 = 1;
    string f2 = 2;
    string f3 = 3;
    string f4 = 4;
}
```

define two comments above the messages that are to be considered rows.  
comments inside, or next to the message declaration are not the right spot.
the comment strings need to be __above__ the message declartions, and only for messages that you want crud operations
generated for.


the comments must start with ```//pserv-```  notice the lack of space between the // and pserv-? This plugin
is not yet smart enough to even know where to trim the spaces. Just follow the examples.  If your use case does not fit in
the examples, then this plugin is probably not for you.


What is generated is this:
```proto
syntax = "proto3";
package test;
import "persist/options.proto";
service Gentest{
	option (persist.service_type) = SPANNER;
	rpc InsertExtras(stream Extra) returns (Extra){
		option (persist.ql) = {
			query:["INSERT INTO extra (f1,f3,f4,f2) VALUES (@f1,@f3,@f4,@f2)"],
		};
	};
	rpc SelectExtraByPk(Extra) returns(Extra){
		option (persist.ql) = {
			query:["SELECT f1,f3,f4,f2) FROM extra WHERE f1=@f1 && f3=@f3 && f4=@f4"],
		};
	};
	rpc DeleteExtra(Extra) returns(Extra){
		option (persist.ql) = {
			query:["DELETE FROM extra VALUES(@f1,@f3,@f4)"],
		};
	};
	rpc UpdateExtra(Extra) returns(Extra){
		option (persist.ql) = {
			query:["UPDATE extra set f2=@f2 PK(f1=@f1,f3=@f3,f4=@f4)"],
		};
	};
	rpc InsertUsers(stream User) returns (User){
		option (persist.ql) = {
			query:["INSERT INTO users (id,name,friends) VALUES (@id,@name,@friends)"],
		};
	};
	rpc SelectUserByPk(User) returns(User){
		option (persist.ql) = {
			query:["SELECT id,name,friends) FROM users WHERE id=@id && name=@name"],
		};
	};
	rpc DeleteUser(User) returns(User){
		option (persist.ql) = {
			query:["DELETE FROM users VALUES(@id,@name)"],
		};
	};
	rpc UpdateUser(User) returns(User){
		option (persist.ql) = {
			query:["UPDATE users set friends=@friends PK(id=@id,name=@name)"],
		};
	};
	rpc InsertFriendss(stream Friends) returns (Friends){
		option (persist.ql) = {
			query:["INSERT INTO friends (logger,names) VALUES (@logger,@names)"],
		};
	};
	rpc SelectFriendsByPk(Friends) returns(Friends){
		option (persist.ql) = {
			query:["SELECT logger,names) FROM friends WHERE logger=@logger"],
		};
	};
	rpc DeleteFriends(Friends) returns(Friends){
		option (persist.ql) = {
			query:["DELETE FROM friends VALUES(@logger)"],
		};
	};
	rpc UpdateFriends(Friends) returns(Friends){
		option (persist.ql) = {
			query:["UPDATE friends set names=@names PK(logger=@logger)"],
		};
	};
}
```

lots of writing we no longer have to do from 3 comments over protobuf messages.


This plugin is not smart enough to generate imports,  or persist type mappings yet.  So you will have to 
1. rename this file (just take out the "generated." part of the filename)
1. add all type mappings
1. add additional queries
1. run protoc again with the persist options.


protoc-gen-persist has documentation on how to work that plugin [HERE](https://github.com/tcncloud/protoc-gen-persist/tree/master)
echo "build *** struct go begin"

CUR_DIR=`pwd`
DIR_NAME=`basename $CUR_DIR`
`cd $CUR_DIR`
protoc --proto_path=./ --gofast_out=:./ message.proto

echo "build *** struct go end"
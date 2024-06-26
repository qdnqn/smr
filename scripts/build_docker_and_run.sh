#!/bin/bash
cd "$(dirname "$0")"
cd ..

echo "Doing work in directory $PWD"

BASE_DIR="$PWD"

for dir in implementations/*/
do
    DIR=${dir%*/}
    DIRNAME="${DIR##*/}"

    cd "$BASE_DIR/implementations/$DIRNAME"
    go build --buildmode=plugin
done

cd "$BASE_DIR"

for dir in operators/*/
do
    DIR=${dir%*/}
    DIRNAME="${DIR##*/}"

    cd "$BASE_DIR/operators/$DIRNAME"
    go build --buildmode=plugin
done

cd "$BASE_DIR"

go build

docker stop $(docker ps -q)
docker rm $(docker ps -aq)
docker build . --file docker/Dockerfile --tag smr:0.0.1
docker run -v /var/run/docker.sock:/var/run/docker.sock -v /tmp:/tmp -v /home/qdnqn/testing-smr:/home/smr-agent/.ssh -p 0.0.0.0:1443:1443 --name smr-agent --dns 127.0.0.1 smr:0.0.1
#! /bin/bash

cd $(dirname $(realpath $0))
./bin/python main.py --category 1 --pages 3 --output hd.xml && aws s3 cp hd.xml s3://libredmm/feeds/4k2/hd.xml --acl public-read
./bin/python main.py --category 3 --pages 3 --output 4k.xml && aws s3 cp 4k.xml s3://libredmm/feeds/4k2/4k.xml --acl public-read

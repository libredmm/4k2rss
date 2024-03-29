#! /usr/bin/env fish

./bin/python main.py --category 1 --pages 5 --output hd.xml && aws s3 cp hd.xml s3://libredmm/feeds/4k2/hd.xml --acl public-read
./bin/python main.py --category 3 --pages 5 --output 4k.xml && aws s3 cp 4k.xml s3://libredmm/feeds/4k2/4k.xml --acl public-read

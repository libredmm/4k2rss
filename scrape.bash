#! /bin/bash

#!/bin/bash

source $DOTFILES/lib/log.sh

PAGE_NUM=3

while getopts ":p:" opt; do
    case $opt in
        p)
            PAGE_NUM=$OPTARG
            ;;
        \?)
            log_fatal "Invalid option: -$OPTARG"
            ;;
    esac
done
shift $((OPTIND-1))

cd $(dirname $(realpath $0))
./bin/python main.py --category 1 --pages $PAGE_NUM --output hd.xml && aws s3 cp hd.xml s3://libredmm/feeds/4k2/hd.xml --acl public-read
./bin/python main.py --category 3 --pages $PAGE_NUM --output 4k.xml && aws s3 cp 4k.xml s3://libredmm/feeds/4k2/4k.xml --acl public-read

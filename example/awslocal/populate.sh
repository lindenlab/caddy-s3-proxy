#!/bin/bash
set -x
awslocal s3 mb s3://my-bucket

echo "hello world" | awslocal s3 cp - s3://my-bucket/hello.txt
set +x


#!/bin/bash
set -x
awslocal s3 mb s3://my-bucket

echo "hello world" | awslocal s3 cp - s3://my-bucket/hello.txt
echo "foo bar" | awslocal s3 cp - s3://my-bucket/foo/index.html
set +x


#!/bin/bash
set -x
awslocal s3 mb s3://my-bucket
echo "hello world" | awslocal s3 cp --no-guess-mime-type --content-type text/html - s3://my-bucket/hello.txt
cat << EOF >  index.html
<!DOCTYPE html><title>s3proxy Examples</title>
<h1>s3proxy Examples</h1>
<p>This is a silly sample site served by caddy and s3proxy.</p>
EOF
awslocal s3 cp index.html s3://my-bucket/index.html

awslocal s3 mb s3://test-results
echo "test reports" | awslocal s3 cp - s3://test-results/index.html
echo "resuults for test 1" | awslocal s3 cp - s3://test-results/1/report.txt
echo "resuults for test 2" | awslocal s3 cp - s3://test-results/2/report.txt
echo "resuults for test 57" | awslocal s3 cp - s3://test-results/57/report.txt
echo "resuults for test 784" | awslocal s3 cp - s3://test-results/784/report.txt
set +x


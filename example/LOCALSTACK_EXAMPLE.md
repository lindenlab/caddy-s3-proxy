
# localstack example

This dir contains an example of using cadd with the caddy-s3-proxy against localstack.  localstack is a mock version of S3 and other AWS services you can run locally.


## Running the demo

You need to have docker and docker-compose installed.

First you need a docker image running caddy with the s3proxy installed.
You can type ```make docker``` to do that.

Then cd into this directory and type:
```
docker-compose up
```

The script awslocal/populate.sh populates the S3 buckets of localstack
with some sample content to so a few examples of how to use s3proxy.
Each of the following examples are configured in the Caddyfile and 
you can play with them by hitting http://localhost.  

## Example #1 - Basic config

Here is a basic config for s3proxy:
```
{
        order s3proxy last
}

s3proxy {
	region "us-west-2"
	bucket "my-bucket"
	index index.html
	endpoint "http://localstack:4566/"
    force_path_style
}
```

With localstack you need to use the endpoint directive.  However, in
normal use on AWS it is not needed.  The bucket directive is required
and with this config any path to caddy is simply used as a key to
fetch content in the bucket.

The region is also required.  (However, it can also be set by setting
the environment variable AWS_REGION.)

Once you have launch docker-compose you can try this config out with the following curl:
```
curl localhost/hello.txt
```
which should return "hello world".


Of course, the content can also be html and seen via a browser.  Open your
browser to *http://localhost* and you should see a sample web page.  This
eample is also utilizing the "index" directive which tells the s3proxy
that if the "key" appears to be a directory - look for an object called
index.html to display instead.

BTW, the default values for the index directive are index.html and index.htm
so the sample use of the index directive here is not even needed.

## Example #2 - Using uri strip_prefix 

Sometimes you do not want all of your sites "path" to be used as a
key into the bucket.  Here is an example where only the last part of
the path is used to create the key in the bucket.  
```
        route /dev/testing/coverage-reports/* {
                uri strip_prefix /dev/testing/coverage-reports
                s3proxy {
                        region "us-west-2"
                        bucket "test-results"
                        endpoint "http://localstack:4566/"
		                force_path_style
                }
        }
```
Here we are using the "uri strip_prefix" directive to strip the path 
/dev/testing/coverage-reports from the url befor calling the s3proxy.
So the web site path of /dev/testing/coverage-reports/2/report.txt will 
return the s3 object with the key of /2/report.txt

Try it out with the following curl:
```
curl localhost/dev/testing/coverage-reports/2/report.txt
```

## Example #3 - using the root directive

Also, the first part of your key may also not be something you want in
your website path.  You can use the root directive to define the "prefix"
to your S3 key that gets prepended to your path before getting an object
from S3.

Here is an example config:
```
        route /animals/* {
                root * /a/long/path/we/have/for
                s3proxy {
                        region "us-west-2"
                        bucket "bkt"
                        endpoint "http://localstack:4566/"
			enable_put
			enable_delete
		                force_path_style
                }
        }
```

You can try it out like this:
```
curl localhost/animals/dog.txt
```

In this case the website request path of /animals/dog.txt
will return the S3 object of /a/long/path/we/have/for/animals/dog.txt


## Example #4 - put and delete operations

It is also possible to send PUT and DELETE operations through the proxy.  Of course,
you will want to ensure this is locked down with proper authentication!  By default,
the operations are not allow - but you can turn support for them on with the
*enable_put* and *enable_delete* directives.

In the above example, these options have been turned on.  You can try it out woth
some curl commands.  (Be sure to set the Content-Type header when loading data.)

```
curl -X PUT -d "COW GOES MOO"  -H "Content-Type: text/plain" localhost/animals/cow.txt
curl localhost/animals/cow.txt

curl -X DELETE  localhost/animals/cow.txt
curl -i localhost/animals/cow.txt
```

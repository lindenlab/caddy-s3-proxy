# caddy-s3-proxy

caddy-s3-proxy allows you to proxy requests directly from S3.

S3 does have the website option, in which case, a normal reverse proxy could be used to display S3 data.
However, it is sometimes inconvient to do that.  This module lets you access S3 data even if website access
is not configured on your bucket.

## Credentials

This module uses the default providor chain to get credentials for access to S3.  This provides several more
secure options to provide credentials for accessing S3 without putting the credentials in the Caddy config.

https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html

## Configuration
The Caddyfile directive would look something like this:
```
	s3proxy {
		bucket <bucket_name>
		region <region_name>
		index  <list of index file names>
	}
```
|  option   |  type  |  default   | help |
|-----------|:------:|------------|------|
| bucket              | string   |                         | S3 bucket |
| endpoint            | string   | aws default             | S3 hostname (optional) |
| region              | string   | env AWS_REGION          | S3 region (optional) |
| index               | string[] | [index.html, index.txt] | Index files to look up for dir path |


## Manipulating the path and resulting S3 key

In general, the path passed in to the module is used as the key to get an object from a bucket.  However,
you may want to serve the S3 data from some other directory in yur web site.  You can do that with the
uri directive.  For example:
```
        route /test-results/* {
                uri strip_prefix /results
                s3proxy {
                        region "us-west-2"
                        bucket "test-results.tilia-inc.com"
                }
        }
```
In this example a web request of *http://www.nysite.com/results/myresults.csv* would request a key from the S3 bucket of */myresults.csv*.
Whereas, if the uri directive was not present it would would request */results/myresults.csv*.

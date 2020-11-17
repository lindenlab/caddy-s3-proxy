package caddys3proxy

import (
	"net/http"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

// Mapping of aws error codes to http status
// See: https://docs.aws.amazon.com/AmazonS3/latest/API/ErrorResponses.html
var awsErrorCodesMapping = map[string]int{
	"AccessDenied":                                   http.StatusForbidden,
	"AccountProblem":                                 http.StatusForbidden,
	"AllAccessDisabled":                              http.StatusForbidden,
	"AmbiguousGrantByEmailAddress":                   http.StatusBadRequest,
	"AuthorizationHeaderMalformed":                   http.StatusBadRequest,
	"BadDigest":                                      http.StatusBadRequest,
	"BucketAlreadyExists":                            http.StatusConflict,
	"BucketAlreadyOwnedByYou":                        http.StatusConflict,
	"BucketNotEmpty":                                 http.StatusConflict,
	"CredentialsNotSupported":                        http.StatusBadRequest,
	"CrossLocationLoggingProhibited":                 http.StatusForbidden,
	"EntityTooSmall":                                 http.StatusBadRequest,
	"EntityTooLarge":                                 http.StatusBadRequest,
	"ExpiredToken":                                   http.StatusBadRequest,
	"IllegalLocationConstraintException":             http.StatusBadRequest,
	"IllegalVersioningConfigurationException":        http.StatusBadRequest,
	"IncompleteBody":                                 http.StatusBadRequest,
	"IncorrectNumberOfFilesInPostRequest":            http.StatusBadRequest,
	"InlineDataTooLarge":                             http.StatusBadRequest,
	"InternalError":                                  http.StatusInternalServerError,
	"InvalidAccessKeyId":                             http.StatusForbidden,
	"InvalidAccessPoint":                             http.StatusBadRequest,
	"InvalidArgument":                                http.StatusBadRequest,
	"InvalidBucketName":                              http.StatusBadRequest,
	"InvalidBucketState":                             http.StatusConflict,
	"InvalidDigest":                                  http.StatusBadRequest,
	"InvalidEncryptionAlgorithmError":                http.StatusBadRequest,
	"InvalidLocationConstraint":                      http.StatusBadRequest,
	"InvalidObjectState":                             http.StatusForbidden,
	"InvalidPart":                                    http.StatusBadRequest,
	"InvalidPartOrder":                               http.StatusBadRequest,
	"InvalidPayer":                                   http.StatusForbidden,
	"InvalidPolicyDocument":                          http.StatusBadRequest,
	"InvalidRange":                                   http.StatusRequestedRangeNotSatisfiable,
	"InvalidRequest":                                 http.StatusBadRequest,
	"InvalidSecurity":                                http.StatusForbidden,
	"InvalidSOAPRequest":                             http.StatusBadRequest,
	"InvalidStorageClass":                            http.StatusBadRequest,
	"InvalidTargetBucketForLogging":                  http.StatusBadRequest,
	"InvalidToken":                                   http.StatusBadRequest,
	"InvalidURI":                                     http.StatusBadRequest,
	"KeyTooLongError":                                http.StatusBadRequest,
	"MalformedACLError":                              http.StatusBadRequest,
	"MalformedPOSTRequest":                           http.StatusBadRequest,
	"MalformedXML":                                   http.StatusBadRequest,
	"MaxMessageLengthExceeded":                       http.StatusBadRequest,
	"MaxPostPreDataLengthExceededError":              http.StatusBadRequest,
	"MetadataTooLarge":                               http.StatusBadRequest,
	"MethodNotAllowed":                               http.StatusMethodNotAllowed,
	"MissingContentLength":                           http.StatusLengthRequired,
	"MissingRequestBodyError":                        http.StatusBadRequest,
	"MissingSecurityElement":                         http.StatusBadRequest,
	"MissingSecurityHeader":                          http.StatusBadRequest,
	"NoLoggingStatusForKey":                          http.StatusBadRequest,
	"NoSuchBucket":                                   http.StatusNotFound,
	"NoSuchBucketPolicy":                             http.StatusNotFound,
	"NoSuchKey":                                      http.StatusNotFound,
	"NoSuchLifecycleConfiguration":                   http.StatusNotFound,
	"NoSuchUpload":                                   http.StatusNotFound,
	"NoSuchVersion":                                  http.StatusNotFound,
	"NotImplemented":                                 http.StatusNotImplemented,
	"NotSignedUp":                                    http.StatusForbidden,
	"OperationAborted":                               http.StatusConflict,
	"PermanentRedirect":                              http.StatusMovedPermanently,
	"PreconditionFailed":                             http.StatusPreconditionFailed,
	"Redirect":                                       http.StatusTemporaryRedirect,
	"RestoreAlreadyInProgress":                       http.StatusConflict,
	"RequestIsNotMultiPartContent":                   http.StatusBadRequest,
	"RequestTimeout":                                 http.StatusBadRequest,
	"RequestTimeTooSkewed":                           http.StatusForbidden,
	"RequestTorrentOfBucketError":                    http.StatusBadRequest,
	"ServerSideEncryptionConfigurationNotFoundError": http.StatusBadRequest,
	"ServiceUnavailable":                             http.StatusServiceUnavailable,
	"SignatureDoesNotMatch":                          http.StatusForbidden,
	"SlowDown":                                       http.StatusServiceUnavailable,
	"TemporaryRedirect":                              http.StatusTemporaryRedirect,
	"TokenRefreshRequired":                           http.StatusBadRequest,
	"TooManyAccessPoints":                            http.StatusBadRequest,
	"TooManyBuckets":                                 http.StatusBadRequest,
	"UnexpectedContent":                              http.StatusBadRequest,
	"UnresolvableGrantByEmailAddress":                http.StatusBadRequest,
	"UserKeyMustBeSpecified":                         http.StatusBadRequest,
	"NoSuchAccessPoint":                              http.StatusBadRequest,
	"InvalidTag":                                     http.StatusBadRequest,
	"MalformedPolicy":                                http.StatusBadRequest,

	// This one is not defined in the doc above - but it happens...
	// https://github.com/aws/aws-sdk-go/issues/3637
	"NotModified": http.StatusNotModified,
}

func convertToCaddyError(err error) caddyhttp.HandlerError {
	caddyErr, isCaddyErr := err.(caddyhttp.HandlerError)
	if isCaddyErr {
		// Already a caddy error
		return caddyErr
	}
	if aerr, ok := err.(awserr.Error); ok {
		//If aws error look up status code in above table
		if code, ok := awsErrorCodesMapping[aerr.Code()]; ok {
			return caddyhttp.Error(code, aerr)
		}
	}

	// Any other error is a 500
	return caddyhttp.Error(http.StatusInternalServerError, err)
}

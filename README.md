s3upload
========

Utility for uploading contents of a directory to an AWS S3 bucket.

This is a very simple program. It performs the following steps:

* List the contents of an S3 bucket
* Scan the files in a directory
* For each file, if the corresponding S3 object does not exist, or is different, upload that file to the S3 bucket

Files are compared using the S3 ETag, which is the MD5 of the contents for S3 objects that have been uploaded in one go.

Usage
=====
`
  s3upload -bucket=bucket-name -dir=/path/to/files
`  

Other command line options:

* `-recursive` Indicates that files in sub-directories should be uploaded as well
* `-include-unknown-mime-types` By default a file with an unknown mime type is not uploaded
* `-verbose` Print additional messages to stderr
* `-help` Print help text to stderr

Environment
===========

This utility expects the following environment variables to be set:

* `AWS_ACCESS_KEY_ID` The AWS access key
* `AWS_SECRET_ACCESS_KEY` The AWS secret key

AWS Permissions
===============

See the `permissions.json` for an example AWS policy that provides the minimum AWS permissions required by this utility.

Limitations
===========

* AWS region is currently hard-coded to ap-southeast-2 (Sydney). Should pick up from the environment and/or command line.

  


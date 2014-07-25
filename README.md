s3upload
========

Utility for uploading contents of a directory to an AWS S3 bucket.

This is a very simple program. It performs the following steps:

* List the contents of an S3 bucket
* Scan the files in a directory
* For each file, if the corresponding S3 object does not exist, or is different, upload that file to the S3 bucket

Files are compared using the S3 ETag, which is the MD5 of the contents for S3 objects that have been uploaded in one go.



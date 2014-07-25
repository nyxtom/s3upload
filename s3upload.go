package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
	"log"
	"mime"
	"os"
	"path"
)

const programName = "s3upload"

// variables set by command line flags
var bucketName string
var baseDir string
var showHelp bool
var verbose bool
var recursive bool
var includeUnknownMimeTypes bool

// contains information about every object in the bucket
// maps the object key name to its etag
var s3Objects = make(map[string]string)

func main() {
	flag.StringVar(&bucketName, "bucket", "", "S3 Bucket Name (required)")
	flag.StringVar(&baseDir, "dir", "", "Local directory (required)")
	flag.BoolVar(&verbose, "verbose", false, "Print extra log messages")
	flag.BoolVar(&showHelp, "help", false, "Show this help")
	flag.BoolVar(&recursive, "recursive", false, "recurse into sub-directories")
	flag.BoolVar(&includeUnknownMimeTypes, "include-unknown-mime-types", false, "upload files with unknown mime types")

	flag.Parse()
	if showHelp {
		fmt.Fprintf(os.Stderr, "usage: %s [ options ]\noptions:\n", programName)
		flag.PrintDefaults()
		return
	}

	if bucketName == "" {
		log.Fatalf("Must specify bucket: use '%s -help' for usage", programName)
	}

	if baseDir == "" {
		log.Fatalf("Must specify directory: use '%s -help' for usage", programName)
	}

	auth, err := aws.EnvAuth()
	if err != nil {
		log.Fatal(err)
	}

	s3Config := s3.New(auth, aws.APSoutheast2)
	bucket := &s3.Bucket{S3: s3Config, Name: bucketName}

	if verbose {
		log.Println("Listing objects in bucket")
	}

	marker := ""
	for {
		listResp, err := bucket.List("", "/", marker, 1000)
		if err != nil {
			log.Fatal(err)
		}

		for _, key := range listResp.Contents {
			s3Objects[key.Key] = key.ETag
			marker = key.Key
		}

		if !listResp.IsTruncated {
			break
		}
		if verbose {
			log.Printf("%d objects loaded", len(s3Objects))
		}
	}

	processDir(baseDir, "", bucket)
}

func processDir(dirName string, s3KeyPrefix string, bucket *s3.Bucket) {
	if verbose {
		log.Printf("Processing directory %s", dirName)
	}

	fileInfos, err := ioutil.ReadDir(dirName)
	if err != nil {
		log.Fatal(err)
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			if recursive {
				subDirName := path.Join(dirName, fileInfo.Name())
				processDir(subDirName, s3KeyPrefix+fileInfo.Name()+"/", bucket)
			}
			continue
		}
		filePath := path.Join(dirName, fileInfo.Name())
		s3Key := s3KeyPrefix + fileInfo.Name()

		putRequired := false
		var data []byte

		s3ETag := s3Objects[s3Key]
		if s3ETag == "" {
			if verbose {
				log.Printf("Not found in S3 bucket: %s", s3Key)
			}
			putRequired = true
		}

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Fatal(err)
		}

		// if the object exists, then we check the MD5 of the file to determine whether
		// the file needs to be uploaded
		if !putRequired {
			digest := md5.Sum(data)
			// note the need to convert digest to a slice because it is a byte array ([16]byte)
			fileETag := "\"" + hex.EncodeToString(digest[:]) + "\""

			if fileETag != s3ETag {
				if verbose {
					log.Printf("Need to upload %s: expected ETag = %s, actual = %s", filePath, fileETag, s3ETag)
				}
				putRequired = true
			}
		}

		if putRequired {
			// TODO: this should be configurable, but for now if the mime-type cannot
			// be determined, do not upload
			contentType := mime.TypeByExtension(path.Ext(fileInfo.Name()))
			if contentType == "" && includeUnknownMimeTypes {
				contentType = "application/octet-stream"
			}

			if contentType != "" {
				err = bucket.Put(s3Key, data, contentType, s3.Private)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Uploaded %s\n", s3Key)
			}

		} else {
			if verbose {
				log.Printf("Identical file, no upload required: %s", filePath)
			}
		}

	}
}

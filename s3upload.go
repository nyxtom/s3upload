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
	"strings"
)

const programName = "s3upload"

// variables set by command line flags
var bucketName string
var dirName string
var showHelp bool
var verbose bool

func main() {
	flag.StringVar(&bucketName, "bucket", "", "S3 Bucket Name (required)")
	flag.StringVar(&dirName, "dir", "", "Local directory (required)")
	flag.BoolVar(&verbose, "verbose", false, "Print extra log messages")
	flag.BoolVar(&showHelp, "help", false, "Show this help")

	flag.Parse()
	if showHelp {
		fmt.Fprintf(os.Stderr, "usage: %s [ options ]\noptions:\n", programName)
		flag.PrintDefaults()
		return
	}

	if bucketName == "" {
		log.Fatalf("Must specify bucket: use '%s -help' for usage", programName)
	}

	if dirName == "" {
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

	// maps key name to etag
	s3Objects := make(map[string]string)
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

	if verbose {
		log.Println("Scanning directory")
	}

	fileInfos, err := ioutil.ReadDir(dirName)
	if err != nil {
		log.Fatal(err)
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			continue
		}
		filePath := path.Join(dirName, fileInfo.Name())

		putRequired := false
		var data []byte

		s3ETag := s3Objects[fileInfo.Name()]
		if s3ETag == "" {
			if verbose {
				log.Printf("Not found in S3 bucket: %s", fileInfo.Name())
			}
			putRequired = true
		}

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Fatal(err)
		}

		if !putRequired {

			digest := md5.Sum(data)
			// note the need to convert digest to a slice because it is a byte array ([16]byte)
			fileETag := "\"" + hex.EncodeToString(digest[:]) + "\""

			if fileETag != s3ETag {
				if verbose {
					log.Printf("Need to upload %s: expected ETag = %s, actual = %s", fileInfo.Name(), fileETag, s3ETag)
				}
				putRequired = true
			}
		}

		if putRequired {
			var contentType string
			if strings.HasSuffix(fileInfo.Name(), ".jpg") {
				contentType = "image/jpeg"
			} else if strings.HasSuffix(fileInfo.Name(), ".gif") {
				contentType = "image/gif"
			} else {
				contentType = mime.TypeByExtension(path.Ext(fileInfo.Name()))
			}

			if contentType != "" {
				err = bucket.Put(fileInfo.Name(), data, contentType, s3.Private)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Uploaded %s\n", fileInfo.Name())
			}

		} else {
			if verbose {
				log.Printf("Identical file, no upload required: %s", fileInfo.Name())
			}
		}

	}
}

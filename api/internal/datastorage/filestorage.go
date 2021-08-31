package datastorage

import (
	"context"
	"log"
	"os"
	"fmt"
	"net/url"
	"path/filepath"
	"io/ioutil"
	"bufio"

	"github.com/equinor/oneseismic/api/api"
	"github.com/equinor/oneseismic/api/internal/message"
)

func NewFileStorage(baseurl string) *FileStorage {
	myUrl, err := url.Parse(baseurl) ; if err != nil {
		panic(fmt.Sprintf("Malformed URL %v", baseurl))
	}
	// verify that the URL points to a file or is a path
	if stat, err := os.Stat(baseurl); err != nil {
		panic(err)
	} else {
		if !stat.IsDir() {
			// path is not a directory
			u, _ := url.ParseRequestURI(baseurl)
			if u.Scheme != "file" {
				panic(fmt.Sprintf("illegal url %v", baseurl))
			} // see https://stackoverflow.com/questions/44294363/what-is-the-idiomatic-way-to-read-urls-with-a-file-scheme-as-filenames-for-readf
			baseurl = u.Path
		}
		// TODO: verify that the directory has the right structure, i.e. that it contains
		// directories "manifests", "fragments", "cubes", etc
	}
	return &FileStorage{baseurl: myUrl}
}
type FileStorage struct {
	baseurl *url.URL
}
func (s FileStorage ) GetUrl() *url.URL { return s.baseurl }

func (s FileStorage) hasAccess(guid string, token string) (bool, error) {
	pth := filepath.Join(s.baseurl.String(), guid+".acl")
	file, err := os.Open(pth) ; if err != nil {
        log.Println(err)
		return false, api.NewNonExistingError(fmt.Sprintf("The resource %v has no ACL", guid))
	}
 
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        if token == scanner.Text() { return true, nil }
    }
 
    if err := scanner.Err(); err != nil {
        log.Println(err)
		return false, api.InternalError{}
	}
	return false, nil
}

func (s FileStorage) FetchManifest(
	ctx context.Context,
	token string,
	guid string,
) ([]byte, error) {
	log.Printf("Requesting manifest for %v using token %v", guid, token)
	ok, err := s.hasAccess(guid, token) ; if err != nil {
		return nil, err
	}

	if !ok {
		return nil, api.NewIllegalAccessError(fmt.Sprintf("The provided token nas no access to %v", guid))
	}
	filepath := filepath.Join(s.baseurl.String(), "manifests", guid+".json")
	content, err := ioutil.ReadFile(filepath)
    if err != nil {
		log.Printf("Failed reading %v", filepath)
		return nil, api.NewNonExistingError(fmt.Sprintf("The resource %v does not exist", guid))
	}
	return []byte(content), nil
}


func (s FileStorage) CreateBlobContainer(task message.Task) (api.AbstractBlobContainer, error) {
	return FileBlobContainer{}, nil
}

type FileBlobContainer struct {
}

func (c FileBlobContainer) ConstructBlobURL(string) api.AbstractBlobURL{
	return FileBlobUrl{}
}

type FileBlobUrl struct {
}

func (u FileBlobUrl) Download(context.Context) ([]byte, error) {
	return nil, nil
}

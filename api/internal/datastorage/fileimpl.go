package datastorage

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/equinor/oneseismic/api/api"
)

func NewFileStorage(kind string, baseurl string) *FileStorage {
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
	}
	return &FileStorage{kind, myUrl}
}
type FileStorage struct {
	kind string
	baseurl *url.URL
}
func (s FileStorage ) GetEndpoint() string { return s.baseurl.String() }
func (s FileStorage ) GetKind() string { return s.kind }

func (s FileStorage) hasAccess(guid string, token string) (bool, error) {
	resourcePath := filepath.Join(s.baseurl.String(), guid)
	if stat, err := os.Stat(resourcePath); err != nil || !stat.IsDir() {
		// path is a directory
        log.Printf("No directory %s", resourcePath)
		return false, api.NewNonExistingError(
			fmt.Sprintf("The resource %s does not exist", guid))

	}

	// If there is no acl, the resource can be read by anyone
	aclPath := filepath.Join(resourcePath, "acl.txt")
	file, err := os.Open(aclPath) ; if err != nil {
		return true, nil
	}

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        if token == scanner.Text() { return true, nil }
    }

    if err := scanner.Err(); err != nil {
        log.Println(err)
		return false, api.NewInternalError(
			fmt.Sprintf("Syntax-error in acl at %s", aclPath))
	}
	return false, nil
}

func (s FileStorage) Get(
	ctx context.Context,
	creds string,
	resource string,
) ([]byte, error) {

	// Parse credentials, split resource and fragment, check access, then reassemble into filepath
	kind, values, err := api.DecodeCredentials(creds)
	if err != nil { return nil, err }
	if kind != "obo" {
		return nil, api.NewIllegalInputError(
						fmt.Sprintf("Expected OBO-creds, got %v", creds))
	}
	user, ok  := values["token"]; if !ok {
		return nil, api.NewIllegalInputError(
						fmt.Sprintf("Expected token-value, got %v", creds))
	}

	resourceParts := strings.Split(resource,"#")
	if len(resourceParts) != 2 {
		return nil, api.NewIllegalInputError(fmt.Sprintf("Bad resource %v", resource))
	}
	urlString := resourceParts[0]
	fragment  := resourceParts[1]
	ok, err = s.hasAccess(urlString, user) ; if err != nil {
		return nil, err
	}
	if !ok {
		return nil, api.NewIllegalAccessError(fmt.Sprintf(
			"The provided token gives no access to %v", resource))
	}
	filepath := filepath.Join(s.baseurl.String(), urlString, fragment)
	content, err := ioutil.ReadFile(filepath)
    if err != nil {
		log.Printf("Failed reading %v", filepath)
		return nil, api.NewNonExistingError(
			fmt.Sprintf("The resource %v does not exist", resource))
	}
	return []byte(content), nil
}

package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

func dieMaybe(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func trimSpaces(str string) string {
	space := regexp.MustCompile(`\s+`)
	return space.ReplaceAllString(str, " ")
}

func getRaw(t *testing.T, url string) []byte {
	resp, err := http.Get(url)
	dieMaybe(t, err)
	body, err := ioutil.ReadAll(resp.Body)
	dieMaybe(t, err)
	return body
}

func getZip(t *testing.T, needle string, dest string) (int, bool) {
	b := getRaw(t, dest)
	unzipped, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	dieMaybe(t, err)
	length := len(unzipped.File)
	for _, file := range unzipped.File {
		if file.Name == needle {
			return length, true
		}
	}
	return length, false
}

func get(t *testing.T, url string) string {
	body := getRaw(t, url)
	return trimSpaces(string(body))
}

func postDummyFile(t *testing.T, url string, path string, payload string) string {
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	body := strings.NewReader("------WebKitFormBoundarycCRIderiXxJWEUcU\r\nContent-Disposition: form-data; name=\"\u1112\u1161 \u1112\u1161\"; filename=\"\u1112\u1161 \u1112\u1161\"\r\nContent-Type: application/octet-stream\r\n\r\n" + payload)
	req, err := http.NewRequest("POST", url+"post", body)
	dieMaybe(t, err)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----WebKitFormBoundarycCRIderiXxJWEUcU")
	req.Header.Set("Gossa-Path", path)

	resp, err := http.DefaultClient.Do(req)
	dieMaybe(t, err)
	defer resp.Body.Close()
	bodyS, err := ioutil.ReadAll(resp.Body)
	dieMaybe(t, err)
	return trimSpaces(string(bodyS))
}

func postJSON(t *testing.T, url string, what string) string {
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(what)))
	dieMaybe(t, err)
	body, err := ioutil.ReadAll(resp.Body)
	dieMaybe(t, err)
	return trimSpaces(string(body))
}

func fetchAndTestDefault(t *testing.T, url string) string {
	body0 := get(t, url)

	if !strings.Contains(body0, `<title>/</title>`) {
		t.Fatal("error title")
	}

	if !strings.Contains(body0, `<h1 onclick="return titleClick(event)">./</h1>`) {
		t.Fatal("error header")
	}

	if !strings.Contains(body0, `href="hols">hols/</a>`) {
		t.Fatal("error hols folder")
	}

	if !strings.Contains(body0, `href="curimit@gmail.com%20%2840%25%29">curimit@gmail.com (40%)/</a>`) {
		t.Fatal("error curimit@gmail.com (40%) folder")
	}

	if !strings.Contains(body0, `href="%E4%B8%AD%E6%96%87">中文/</a>`) {
		t.Fatal("error 中文 folder")
	}

	if !strings.Contains(body0, `href="custom_mime_type.types">custom_mime_type.types</a>`) {
		t.Fatal("error row custom_mime_type")
	}

	return body0
}

func doTestRegular(t *testing.T, url string, testExtra bool) {
	var payload, path, body0, body1, body2 string

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test fetching default path")
	fetchAndTestDefault(t, url)

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test fetching another page")
	body0 = get(t, url+"/hols")
	if !strings.Contains(body0, "glasgow.jpg") {
		t.Fatal("fetching a subfolder failed")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test fetching an invalid path - redirected to root")
	fetchAndTestDefault(t, url+"../../")
	fetchAndTestDefault(t, url+"hols/../../")

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test fetching regular files")
	body0 = get(t, url+"subdir_with%20space/file_with%20space.html")
	body1 = get(t, url+"fancy-path/a")
	if body0 != `<b>spacious!!</b> ` || body1 != `fancy! ` {
		t.Fatal("fetching a regular file errored")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test fetching a invalid file")
	path = "../../../../../../../../../../etc/passwd"
	if !testExtra && get(t, url+path) != `error` {
		t.Fatal("fetching a invalid file didnt errored")
	} else if testExtra {
		fetchAndTestDefault(t, url+path) // extra path will just redirect to root dir
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test zipping of folder 中文")
	len, foundFile := getZip(t, "檔案.html", url+"zip?zipPath=%2F%E4%B8%AD%E6%96%87%2F&zipName=%E4%B8%AD%E6%96%87")
	if len != 1 || !foundFile {
		t.Fatal("invalid zip generated")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test zipping of folder with hidden file")
	_, foundHidden := getZip(t, ".hidden-folder/some-file", url+"zip?zipPath=%2fhols%2f&zipName=hols")
	if foundHidden && !testExtra {
		t.Fatal("invalid zip generated - shouldnt contain hidden folder")
	} else if !foundHidden && testExtra {
		t.Fatal("invalid zip generated - should contain hidden folder")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test zip invalid path")
	body0 = get(t, url+"zip?zipPath=%2Ftmp&zipName=subdir")
	println(body0)
	if body0 != `error` {
		t.Fatal("zip passed for invalid path")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test mkdir rpc")
	body0 = postJSON(t, url+"rpc", `{"call":"mkdirp","args":["/AAA"]}`)
	if body0 != `ok` {
		t.Fatal("mkdir rpc errored")
	}

	body0 = fetchAndTestDefault(t, url)
	if !strings.Contains(body0, `href="AAA">AAA/</a>`) {
		t.Fatal("mkdir rpc folder not created")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test invalid mkdir rpc")
	body0 = postJSON(t, url+"rpc", `{"call":"mkdirp","args":["../BBB"]}`)
	if body0 != `error` {
		t.Fatal("invalid mkdir rpc didnt errored #0")
	}

	body0 = postJSON(t, url+"rpc", `{"call":"mkdirp","args":["/../BBB"]}`)
	if body0 != `error` {
		t.Fatal("invalid mkdir rpc didnt errored #1")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test post file")
	path = "%2F%E1%84%92%E1%85%A1%20%E1%84%92%E1%85%A1" // "하 하" encoded
	payload = "123 하"
	body0 = postDummyFile(t, url, path, payload)
	body1 = get(t, url+path)
	body2 = fetchAndTestDefault(t, url)
	if body0 != `ok` || body1 != payload || !strings.Contains(body2, `href="%E1%84%92%E1%85%A1%20%E1%84%92%E1%85%A1">하 하</a>`) {
		t.Fatal("post file errored")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test post file incorrect path")
	body0 = postDummyFile(t, url, "%2E%2E"+path, payload)
	if !strings.Contains(body0, `err`) {
		t.Fatal("post file incorrect path didnt errored")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test post file")
	path = "2024-01-02-10:36:58.png"
	payload = "123123123123123123123123"
	body0 = postDummyFile(t, url, path, payload)
	body1 = get(t, url+path)
	body2 = fetchAndTestDefault(t, url)
	if body0 != `ok` || body1 != payload || !strings.Contains(body2, `href="2024-01-02-10:36:58.png"`) {
		t.Fatal("post file errored")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test mv rpc")
	body0 = postJSON(t, url+"rpc", `{"call":"mv","args":["/AAA", "/hols/AAA"]}`)
	body1 = fetchAndTestDefault(t, url)
	if body0 != `ok` || strings.Contains(body1, `href="AAA">AAA/</a></td> </tr>`) {
		t.Fatal("mv rpc errored")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test upload in new folder")
	payload = "test"
	body0 = postDummyFile(t, url, "%2Fhols%2FAAA%2Fabcdef", payload)
	body1 = get(t, url+"hols/AAA/abcdef")
	if body0 != `ok` || body1 != payload {
		t.Fatal("upload in new folder errored")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test symlink, should succeed: ", testExtra)
	body0 = get(t, url+"/support/")
	hasListing := strings.Contains(body0, `readme.md`)
	body1 = get(t, url+"/support/readme.md")
	hasReadme := strings.Contains(body1, `the master branch is automatically built and pushed`)
	body2 = get(t, url)
	hasMainListing := strings.Contains(body2, `href="support">support/</a>`)

	if !testExtra && hasReadme {
		t.Fatal("error symlink file reached where illegal")
	} else if testExtra && !hasReadme {
		t.Fatal("error symlink file unreachable")
	}
	if !testExtra && hasListing {
		t.Fatal("error symlink folder reached where illegal")
	} else if testExtra && !hasListing {
		t.Fatal("error symlink folder unreachable")
	}
	if !testExtra && hasMainListing {
		t.Fatal("error symlink folder where illegal")
	} else if testExtra && !hasMainListing {
		t.Fatal("error symlink folder unreachable")
	}

	if testExtra {
		fmt.Println("\r\n~~~~~~~~~~ test symlink mkdir & cleanup")
		body0 = postJSON(t, url+"rpc", `{"call":"mkdirp","args":["/support/testfolder"]}`)
		if body0 != `ok` {
			t.Fatal("error symlink mkdir")
		}

		body0 = postJSON(t, url+"rpc", `{"call":"rm","args":["/support/testfolder"]}`)
		if body0 != `ok` {
			t.Fatal("error symlink rm")
		}
	}

	fmt.Println("\r\n~~~~~~~~~~ test hidden file, should succeed: ", testExtra)
	body0 = get(t, url+"/.testhidden")
	hasHidden := strings.Contains(body0, `test`)
	if !testExtra && hasHidden {
		t.Fatal("error hidden file reached where illegal")
	} else if testExtra && !hasHidden {
		t.Fatal("error hidden file unreachable")
	}

	//
	fmt.Println("\r\n~~~~~~~~~~ test upload in new folder")
	payload = "test"
	body0 = postDummyFile(t, url, "%2Fhols%2FAAA%2Fabcdef", payload)
	body1 = get(t, url+"hols/AAA/abcdef")
	if body0 != `ok` || body1 != payload {
		t.Fatal("upload in new folder errored")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test rm rpc & cleanup")
	body0 = postJSON(t, url+"rpc", `{"call":"rm","args":["/hols/AAA"]}`)
	if body0 != `ok` {
		t.Fatal("cleanup errored #0")
	}

	body0 = get(t, url+"hols/AAA")
	if !strings.Contains(body0, `error`) {
		t.Fatal("cleanup errored #1")
	}

	body0 = postJSON(t, url+"rpc", `{"call":"rm","args":["/하 하"]}`)
	if body0 != `ok` {
		t.Fatal("cleanup errored #2")
	}

	fmt.Printf("\r\n=========\r\n")
}

func doTestReadonly(t *testing.T, url string) {
	var payload, path, body0, body1 string

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test fetching default path")
	fetchAndTestDefault(t, url)

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test fetching an invalid path - redirected to root")
	fetchAndTestDefault(t, url+"../../")
	fetchAndTestDefault(t, url+"hols/../../")

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test fetching regular files")
	body0 = get(t, url+"subdir_with%20space/file_with%20space.html")
	body1 = get(t, url+"fancy-path/a")
	if body0 != `<b>spacious!!</b> ` || body1 != `fancy! ` {
		t.Fatal("fetching a regular file errored")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test fetching a invalid file")
	path = "../../../../../../../../../../etc/passwd"
	if get(t, url+path) != `error` {
		t.Fatal("fetching a invalid file didnt errored")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test mkdir rpc")
	body0 = postJSON(t, url+"rpc", `{"call":"mkdirp","args":["/AAA"]}`)
	if body0 == `ok` {
		t.Fatal("mkdir rpc passed - should not be allowed")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test post file")
	path = "%2F%E1%84%92%E1%85%A1%20%E1%84%92%E1%85%A1" // "하 하" encoded
	payload = "123 하"
	body0 = postDummyFile(t, url, path, payload)
	get(t, url+path)
	if body0 == `ok` {
		t.Fatal("post file passed - should not be allowed")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test mv rpc")
	body0 = postJSON(t, url+"rpc", `{"call":"mv","args":["/AAA", "/hols/AAA"]}`)
	fetchAndTestDefault(t, url)
	if body0 == `ok` {
		t.Fatal("mv rpc passed - should not be allowed")
	}

	// ~~~~~~~~~~~~~~~~~
	fmt.Println("\r\n~~~~~~~~~~ test rm rpc & cleanup")
	body0 = postJSON(t, url+"rpc", `{"call":"rm","args":["/hols/AAA"]}`)
	if body0 == `ok` {
		t.Fatal("cleanup passed - should not be allowed")
	}

	fmt.Printf("\r\n=========\r\n")
}

func TestNormal(t *testing.T) {
	fmt.Println("========== testing normal path ============")
	doTestRegular(t, "http://127.0.0.1:8001/", false)
}

func TestExtra(t *testing.T) {
	fmt.Println("========== testing extras options ============")
	doTestRegular(t, "http://127.0.0.1:8001/fancy-path/", true)
}

func TestRo(t *testing.T) {
	fmt.Println("========== testing read only ============")
	doTestReadonly(t, "http://127.0.0.1:8001/")
}

func TestRunMain(t *testing.T) {
	main()
}

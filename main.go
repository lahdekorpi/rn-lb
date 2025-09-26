package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	// google url

	url := "http://google.com"

	//get request

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching the URL:", err)
		return
	}
	defer resp.Body.Close()

	//status code

	fmt.Println("status code:", resp.StatusCode)

	//read body

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading the response body:", err)
		return
	}
	//print first 500 chars of body
	fmt.Println("response body(first 500chars):")
	fmt.Println(string(body[:500]))
}

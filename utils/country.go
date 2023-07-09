package utils

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func GetCountryCode(peer Peer) string {
	apiUrl := fmt.Sprintf("https://ipapi.co/%s/country", peer.IP.String())

	httpResp, err := http.Get(apiUrl)
	if err != nil {
		return "CA"
	}

	if httpResp.StatusCode != http.StatusOK {
		return "CA"
	}

	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return string(bodyBytes)
}

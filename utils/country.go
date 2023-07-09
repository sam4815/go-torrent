package utils

import (
	"fmt"
	"io"
	"net/http"
)

func GetCountryCode(peer Peer) string {
	apiUrl := fmt.Sprintf("https://ipapi.co/%s/country", peer.IP.String())

	client := http.Client{}
	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return "CA"
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/113.0")

	httpResp, err := client.Do(req)
	if err != nil {
		return "CA"
	}

	if httpResp.StatusCode != http.StatusOK {
		return "CA"
	}

	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "CA"
	}

	return string(bodyBytes)
}

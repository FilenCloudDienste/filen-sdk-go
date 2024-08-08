package client

import (
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
)

var (
	egestUrls = []string{
		"https://egest.filen.io",
		"https://egest.filen.net",
		"https://egest.filen-1.net",
		"https://egest.filen-2.net",
		"https://egest.filen-3.net",
		"https://egest.filen-4.net",
		"https://egest.filen-5.net",
		"https://egest.filen-6.net",
	}
)

func (client *Client) DownloadFileChunk(uuid string, region string, bucket string, chunk int) ([]byte, error) {
	egestUrl := egestUrls[rand.IntN(len(egestUrls))]
	url := fmt.Sprintf("%s/%s/%s/%s/%v", egestUrl, region, bucket, uuid, chunk)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+client.APIKey)
	httpClient := http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

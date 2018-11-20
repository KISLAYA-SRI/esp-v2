// Copyright 2018 Google Cloud Platform Proxy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configmanager

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	serviceAccountTokenURL = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token"
	timeNow                = time.Now
)

// metadata updates and stores Metadata from GCE.
var metadata struct {
	accessToken  string
	tokenTimeout time.Time
}

type metadataTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

var metadataClient http.Client

var getMetadata = func(path string) ([]byte, error) {
	req, _ := http.NewRequest("GET", path, nil)
	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := metadataClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func fetchAccessToken() (string, time.Duration, error) {
	now := timeNow()

	// Follow the similar logic as GCE metadata server, where returned token will be valid for at least 60s.
	if metadata.accessToken != "" && !now.After(metadata.tokenTimeout.Add(-time.Second*60)) {
		return metadata.accessToken, metadata.tokenTimeout.Sub(now), nil
	}

	tokenBody, err := getMetadata(serviceAccountTokenURL)
	if err != nil {
		return "", 0, err
	}

	var resp metadataTokenResponse
	if err = json.Unmarshal(tokenBody, &resp); err != nil {
		return "", 0, err
	}

	expires := time.Duration(resp.ExpiresIn) * time.Second
	metadata.accessToken = resp.AccessToken
	metadata.tokenTimeout = now.Add(expires)
	return metadata.accessToken, expires, nil
}
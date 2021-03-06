// Copyright © 2018 SENETAS SECURITY PTY LTD
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

package auth

import (
	"net/http"
	"net/url"
	"regexp"

	"github.com/Senetas/crypto-cli/registry/httpclient"
	"github.com/Senetas/crypto-cli/utils"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/api/v2"
	dregistry "github.com/docker/docker/registry"
	"github.com/pkg/errors"
)

var challengeRE = regexp.MustCompile(`^\s*Bearer\s+realm="([^"]+)",service="([^"]+)"(,scope="([^"]+)")?\s*$`)

// Challenge from a auth server
type Challenge struct {
	realm   *url.URL
	service string
	scope   string
}

// ParseChallengeHeader parses the challenge header and extract the relevant parts
func ParseChallengeHeader(header string) (ch *Challenge, err error) {
	match := challengeRE.FindAllStringSubmatch(header, -1)

	if len(match) != 1 {
		err = errors.Errorf("malformed challenge header: %s", header)
		return
	}

	ch = &Challenge{
		service: match[0][2],
		scope:   match[0][4],
	}

	ch.realm, err = url.Parse(match[0][1])
	if err != nil {
		err = errors.WithStack(err)
		return
	}

	return
}

// buildURL creates the url to respond to the challenge
func (c *Challenge) buildURL() *url.URL {
	authURL := *c.realm
	authParams := make(url.Values)
	authParams.Set("service", c.service)
	if c.scope != "" {
		authParams.Set("scope", c.scope)
	}
	authURL.RawQuery = authParams.Encode()
	return &authURL
}

// ChallengeHeader requests the challenge header from the auth server
func ChallengeHeader(
	ref reference.Named,
	repoInfo dregistry.RepositoryInfo,
	endpoint dregistry.APIEndpoint,
	creds Credentials,
) (auth string, err error) {
	bldr := v2.NewURLBuilder(endpoint.URL, false)

	urlStr, err := bldr.BuildManifestURL(ref)
	if err != nil {
		err = errors.Wrapf(err, "base = %s", endpoint.URL)
		return
	}

	req, err := http.NewRequest("PUT", urlStr, nil)
	if err != nil {
		return
	}

	resp, err := httpclient.DoRequest(httpclient.DefaultClient, req, true, true)
	if resp != nil {
		defer func() { err = utils.CheckedClose(resp.Body, err) }()
	}
	if err != nil {
		return
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		auth = resp.Header.Get("Www-Authenticate")
		if auth == "" {
			err = errors.New("login error")
		}
	case http.StatusOK:
	default:
		err = errors.New("login not supported")
	}
	return
}

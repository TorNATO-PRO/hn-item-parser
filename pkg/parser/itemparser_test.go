// BSD 3-Clause License
//
// Copyright (c) 2024, Nathan Waltz
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//	list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	this list of conditions and the following disclaimer in the documentation
//	and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its
//	contributors may be used to endorse or promote products derived from
//	this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package parser_test

import (
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TorNATO-PRO/hn-item-parser/v2/pkg/model"
	"github.com/TorNATO-PRO/hn-item-parser/v2/pkg/parser"
	"github.com/stretchr/testify/assert"
)

// dateLayout specifies the date layout constant to use.
const dateLayout = "2006-01-02T15:04:05"

// TestParserSamples tests HTML files that were derived from
// calling HN.
func TestParserSamples(t *testing.T) {
	type TestDef struct {
		Title       model.Title
		Author      string
		Date        time.Time
		ID          int
		Points      int
		NumComments int
		Testfile    string
		Testname    string
	}

	referenceOne, err := url.Parse("https://github.com/glenjamin/node-fib")

	assert.Nil(t, err)

	dateOne, err := time.Parse(dateLayout, "2011-10-03T18:32:05")

	tests := []TestDef{
		{
			Title: model.Title{
				Name:      "Node-fib: Fast non-blocking fibonacci server",
				Reference: referenceOne,
			},
			Author:      "dchest",
			Testfile:    filepath.Join("testdata", "sample1.html"),
			NumComments: 118,
			Points:      194,
			ID:          3067403,
			Date:        dateOne,
			Testname:    "TestSampleOne",
		},
	}

	for _, test := range tests {
		t.Run(test.Testname, func(t *testing.T) {
			sample, err := os.ReadFile(test.Testfile)

			assert.Nil(t, err)

			reader := bytes.NewReader(sample)

			parsed, err := parser.ParseHTML(reader)

			assert.Nil(t, err)

			assert.Equal(t, test.Title, parsed.Title)

			assert.Equal(t, test.Author, parsed.Author)

			assert.Equal(t, test.Points, parsed.Points)

			assert.Equal(t, test.ID, parsed.ID)

			assert.Equal(t, test.Date, parsed.Date)

			assert.Equal(t, test.NumComments, len(parsed.Comments))
		})
	}

}

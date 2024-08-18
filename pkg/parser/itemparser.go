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

// Package parser implements a parser for the HackerNews items.
package parser

import (
	"bytes"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/TorNATO-PRO/hn-item-parser/v2/pkg/model"
	"golang.org/x/net/html"
)

// dateLayout specifies the date layout constant to use.
const dateLayout = "2006-01-02T15:04:05"

// ParseHTML parses an HTML document from the provided io.Reader and populates
// a model.Item struct with the relevant data extracted from the document.
// It returns a pointer to the populated model.Item and an error if parsing
// fails or if any issues occur during the node traversal process.
func ParseHTML(doc io.Reader) (*model.Item, error) {
	var item model.Item

	node, err := html.Parse(doc)
	if err != nil {
		return nil, err
	}

	err = nodeTraverser(node, &item)

	return &item, err
}

// nodeTraverser recursively traverses an HTML node tree, processing each node
// that meets specific criteria and populating the provided model.Item struct
// with the relevant data. The function returns an error if any issues occur
// during the traversal or processing of nodes.
func nodeTraverser(node *html.Node, item *model.Item) error {
	if node.Type == html.ElementNode && shouldProcess(node) {
		err := processNode(node, item)

		if err != nil {
			return err
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		err := nodeTraverser(child, item)

		if err != nil {
			return err
		}
	}

	return nil
}

// shouldProcess checks if a given HTML node is one of the specified element types
// ("td", "tr", "span", "a", "table") that should be processed for data extraction.
// Returns true if the node matches one of these types, false otherwise.
func shouldProcess(node *html.Node) bool {
	return node.Data == "td" ||
		node.Data == "tr" ||
		node.Data == "span" ||
		node.Data == "a" ||
		node.Data == "table"
}

// processNode processes a given HTML node to extract and populate various fields
// of a model.Item struct, such as the title, ID, score, date, author, and comments.
// The function returns an error if any of the extraction operations fail.
func processNode(node *html.Node, item *model.Item) error {
	// process the title
	if err := extractTitle(node, item); err != nil {
		return err
	}

	// process the ID
	if err := extractID(node, item); err != nil {
		return err
	}

	// the subline parent contains all of the
	// score, date, and author
	if classIs(node.Parent, "subline") {
		// process the score
		if err := extractScore(node, item); err != nil {
			return err
		}

		// process the date
		if err := extractDate(node, item); err != nil {
			return err
		}

		// process the author
		if err := extractAuthor(node, item); err != nil {
			return err
		}
	}

	// this is where the comments lie
	if classIs(node, "comment-tree") {
		// process the comments
		extractComments(node, item)
	}

	return nil
}

// extractComments traverses an HTML node tree to extract and parse comments within
// a "comment-tree" structure, populating the provided model.Item with a list of
// model.Comment structs. Returns an error if any issues arise during comment extraction.
func extractComments(node *html.Node, item *model.Item) error {
	if node == nil || node.FirstChild == nil || !classIs(node, "comment-tree") {
		return nil
	}

	var comments []model.Comment

	commentChild := getChildRefByClass(node, "athing comtr")

	if commentChild == nil {
		return nil
	}

	// make sure we are scanned to the exact one
	for commentChild.PrevSibling != nil {
		commentChild = commentChild.PrevSibling
	}

	for child := commentChild; child != nil; child = child.NextSibling {
		comment, err := extractComment(child)

		if err != nil {
			return err
		}

		if comment != nil {
			comments = append(comments, *comment)
		}
	}

	item.Comments = comments

	return nil
}

// extractComment extracts and parses a single comment from an HTML node, populating
// a model.Comment struct with the relevant data such as ID, author, date, parent ID,
// and content. Returns a pointer to the populated model.Comment and an error if any
// issues occur during the parsing process.
func extractComment(node *html.Node) (*model.Comment, error) {
	var comment model.Comment

	if node == nil || !classIs(node, "athing comtr") {
		return nil, nil
	}

	if err := extractCommentID(node, &comment); err != nil {
		return nil, err
	}

	// scan to here to improve efficiency
	defaultNode := getChildRefByClass(node, "default")

	if defaultNode == nil {
		return nil, nil
	}

	if err := extractCommentAuthor(node, &comment); err != nil {
		return nil, err
	}

	if err := extractCommentDate(node, &comment); err != nil {
		return nil, err
	}

	if err := extractParentID(node, &comment); err != nil {
		return nil, err
	}

	if err := extractContent(node, &comment); err != nil {
		return nil, err
	}

	return &comment, nil
}

// extractCommentID extracts the comment ID from the provided HTML node and assigns it
// to the model.Comment struct. Returns an error if the ID cannot be parsed.
func extractCommentID(node *html.Node, comment *model.Comment) error {
	if node == nil || !classIs(node, "athing comtr") {
		return nil
	}

	idString := getAttr(node, "id")

	id, err := strconv.Atoi(idString)

	if err != nil {
		return err
	}

	comment.ID = id

	return nil
}

// extractParentID extracts the parent ID of a comment, if it exists.
func extractParentID(node *html.Node, comment *model.Comment) error {
	parentNode := getChildRefByData(node, "parent")

	if parentNode == nil {
		return nil
	}

	parent := parentNode.Parent

	ref := getAttr(parent, "href")

	if ref == "" {
		return nil
	}

	pid, err := strconv.Atoi(ref[1:])

	if err != nil {
		return err
	}

	comment.ParentID = &pid

	return nil
}

// extractContent extracts the content of a comment from the provided HTML node and
// assigns it to the model.Comment struct. Returns an error if content extraction fails.
func extractContent(node *html.Node, comment *model.Comment) error {
	contentNode := getChildRefByClass(node, "commtext c00")

	if contentNode == nil {
		return nil
	}

	var buf bytes.Buffer

	err := html.Render(&buf, contentNode)

	if err != nil {
		return err
	}

	comment.Content = fixText(buf.String())

	return nil
}

// extractCommentAuthor extracts the author's name from the provided HTML node and
// assigns it to the model.Comment struct. Returns nil if the author cannot be found.
func extractCommentAuthor(node *html.Node, comment *model.Comment) error {
	ref := getChildRefByClass(node, "hnuser")

	if ref == nil || ref.FirstChild == nil {
		return nil
	}

	comment.Author = ref.FirstChild.Data

	return nil
}

// extractCommentDate extracts and parses the date of the comment from the provided
// HTML node and assigns it to the model.Comment struct. Returns an error if the
// date cannot be parsed.
func extractCommentDate(node *html.Node, comment *model.Comment) error {
	ref := getChildRefByClass(node, "age")

	if ref == nil {
		return nil
	}

	titleString := getAttr(ref, "title")

	posted, err := time.Parse(dateLayout, titleString)

	if err != nil {
		return err
	}

	comment.Date = posted

	return nil
}

// isPageSpace checks whether the provided HTML node represents a "pagespace" <tr> tag.
// Returns true if the node has an ID attribute with the value "pagespace", false otherwise.
func isPageSpace(node *html.Node) bool {
	for _, attr := range node.Attr {
		if attr.Key == "id" && attr.Val == "pagespace" {
			return true
		}
	}

	return false
}

// extractTitle extracts the title and its reference URL from the provided HTML node
// and assigns them to the model.Item struct. Returns an error if the title or URL
// cannot be extracted or parsed.
func extractTitle(node *html.Node, item *model.Item) error {
	// if you are new to Go, then you should know that
	// Go really hates cyclomatic complexity and nested
	// if statements.

	if node == nil || node.Data != "td" {
		return nil
	}

	hasTitleClass := getAttr(node, "class") == "title"

	// if a title class doesn't even exist,
	// then don't waste anymore time
	if !hasTitleClass {
		return nil
	}

	spanChild := node.FirstChild

	if spanChild == nil || spanChild.Data != "span" {
		return nil
	}

	hasTitleLine := getAttr(spanChild, "class") == "titleline"

	// if a titleline class doesn't exist,
	// don't waste anymore time
	if !hasTitleLine {
		return nil
	}

	aChild := spanChild.FirstChild

	if aChild == nil || aChild.Data != "a" {
		return nil
	}

	item.Title.Name = fixText(aChild.FirstChild.Data)

	// find the reference
	href := getAttr(aChild, "href")

	reference, err := url.Parse(href)

	if err != nil {
		return err
	}

	item.Title.Reference = reference

	return nil
}

// extractScore extracts and parses the score from the provided HTML node and assigns it
// to the model.Item struct. Returns an error if the score cannot be parsed.
func extractScore(node *html.Node, item *model.Item) error {
	if node == nil || node.Data != "span" {
		return nil
	}

	hasScore := getAttr(node, "class") == "score"

	if !hasScore {
		return nil
	}

	if node.FirstChild == nil {
		return nil
	}

	scoreText := fixText(node.FirstChild.Data)

	scoreSlice := strings.Split(scoreText, " ")

	if len(scoreSlice) != 2 {
		return nil
	}

	points, err := strconv.Atoi(scoreSlice[0])

	if err != nil {
		return err
	}

	item.Points = points

	return nil
}

// extractDate extracts and parses the date of the item from the provided HTML node
// and assigns it to the model.Item struct. Returns an error if the date cannot be parsed.
func extractDate(node *html.Node, item *model.Item) error {
	if node == nil || node.Data != "span" {
		return nil
	}

	hasDate := classIs(node, "age")

	if !hasDate {
		return nil
	}

	titleString := getAttr(node, "title")

	posted, err := time.Parse(dateLayout, titleString)

	if err != nil {
		return err
	}

	item.Date = posted

	return nil
}

// extractAuthor extracts the author's name from the provided HTML node and assigns it
// to the model.Item struct. Returns nil if the author cannot be found.
func extractAuthor(node *html.Node, item *model.Item) error {
	if node != nil && classIs(node, "hnuser") && node.FirstChild != nil {
		author := fixText(node.FirstChild.Data)

		item.Author = author
	}

	return nil
}

// extractID extracts and parses the ID of the item from the provided HTML node and
// assigns it to the model.Item struct. Returns an error if the ID cannot be parsed.
func extractID(node *html.Node, item *model.Item) error {
	if node != nil && classIs(node, "athing") && node.FirstChild != nil {
		idString := getAttr(node, "id")

		id, err := strconv.Atoi(idString)

		if err != nil {
			return err
		}

		item.ID = id
	}

	return nil
}

// fixText removes any extraneous whitespace from the provided text string to ensure
// the text is clean and free of unnecessary spaces. Returns the cleaned text string.
func fixText(text string) string {
	regex := regexp.MustCompile(`\s+`)
	strs := regex.Split(text, -1)
	return strings.Join(strs, " ")
}

// getAttr retrieves the value of the specified attribute from the provided HTML node.
// Returns the attribute value as a string, or an empty string if the attribute is not found.
func getAttr(node *html.Node, attr string) string {
	for _, att := range node.Attr {
		if attr == att.Key {
			return att.Val
		}
	}

	return ""
}

// hasChildClass checks whether the provided HTML node has a child node with the
// specified class. Returns true if a matching child node is found, false otherwise.
func hasChildClass(node *html.Node, class string) bool {
	return getChildRefByClass(node, class) != nil
}

// getChildRefByClass recursively searches for and returns the first child node
// of the provided HTML node that matches the specified class. Returns nil if no
// matching child node is found.
func getChildRefByClass(node *html.Node, class string) *html.Node {
	return getChildRefByPredicate(node, func(n *html.Node) bool {
		return classIs(n, class)
	})
}

// getChildRefByID recursively searches for and returns the first child node
// of the provided HTML node that matches the specified ID. Returns nil if no
// matching child node is found.
func getChildRefByID(node *html.Node, id string) *html.Node {
	return getChildRefByPredicate(node, func(n *html.Node) bool {
		return getAttr(node, "id") == id
	})
}

// getChildRefByData recursively searches for and returns the first child node
// of the provided HTML node that matches the specified data. Returns nil if no
// matching child node is found.
func getChildRefByData(node *html.Node, data string) *html.Node {
	return getChildRefByPredicate(node, func(n *html.Node) bool {
		return n.Data == data
	})
}

// getChildRefByData recursively searches for and returns the first child node
// of the provided HTML node that matches the specified predicate. Returns nil if no
// matching child node is found.
func getChildRefByPredicate(node *html.Node, predicate func(*html.Node) bool) *html.Node {
	if node == nil {
		return nil
	}

	if predicate(node) {
		return node
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if result := getChildRefByPredicate(child, predicate); result != nil {
			return result
		}
	}

	return nil
}

// classIs checks whether the provided HTML node belongs to the specified class.
// Returns true if the node's class matches the specified class, false otherwise.
func classIs(node *html.Node, class string) bool {
	if node == nil {
		return false
	}

	return getAttr(node, "class") == class
}

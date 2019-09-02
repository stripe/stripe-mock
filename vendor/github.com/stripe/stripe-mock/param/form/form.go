// Package form is a small package used to hold a few types common to the param
// package so that we don't have import cycles.
package form

//
// Public types
//

// Pair is a key/value pair as extracted from a form-encoded string. For
// example, "a=b" is the pair [a, b].
type Pair [2]string

// Values is a full slice of all the key/value pairs from a form-encoded
// string.
type Values []Pair

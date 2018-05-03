// Package form is a small package used to hold a few types common to the param
// package so that we don't have import cycles.
package form

//
// Public types
//

// FormPair is a key/value pair as extracted from a form-encoded string. For
// example, "a=b" is the pair [a, b].
type FormPair [2]string

// FormValues is a full slice of all the key/value pairs from a form-encoded
// string.
type FormValues []FormPair

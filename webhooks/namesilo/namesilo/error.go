// SPDX-License-Identifier: MIT

package namesilo

import "errors"

var (
	ErrNamesiloHTTPNotOK = errors.New("bad http status code")
	ErrAPIKeyDecode      = errors.New("error decoding api key")
	ErrTXTRecordCreate   = errors.New("TXT record could not be created")
	ErrTXTRecordFetch    = errors.New("TXT record fetch failed")
	ErrTXTRecordDelete   = errors.New("TXT record delete failed")
)

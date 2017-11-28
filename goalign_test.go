package main

import (
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"bytes"

	"github.com/stretchr/testify/require"
)

func TestAlign(t *testing.T) {
	infos, err := ioutil.ReadDir("test_files")
	require.NoError(t, err)

	results := make(map[string]string)
	solutions := make(map[string]string)

	for _, info := range infos {
		baseFilename := strings.SplitN(info.Name(), ".", 2)[0]
		if path.Ext(info.Name()) == "sol" {
			solution, err := ioutil.ReadFile(path.Join("test_files", info.Name()))
			require.NoError(t, err)
			solutions[baseFilename] = string(solution)
		} else {
			buf := bytes.NewBufferString("")
			walker := getWalker(buf)
			err := walker(path.Join("test_files", info.Name()), info, nil)
			require.NoError(t, err)
			results[baseFilename] = buf.String()
		}
	}
	for key, result := range results {
		require.Equal(t, solutions[key], result)
	}
}

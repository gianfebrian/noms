// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/clienttest"
	"github.com/attic-labs/testify/suite"
)

func TestFetch(t *testing.T) {
	suite.Run(t, &testSuite{})
}

type testSuite struct {
	clienttest.ClientTestSuite
}

func (s *testSuite) TestImportFromStdin() {
	assert := s.Assert()

	oldStdin := os.Stdin
	newStdin, blobOut, err := os.Pipe()
	assert.NoError(err)

	os.Stdin = newStdin
	defer func() {
		os.Stdin = oldStdin
	}()

	go func() {
		blobOut.Write([]byte("abcdef"))
		blobOut.Close()
	}()

	dsName := spec.CreateValueSpecString("ldb", s.LdbDir, "ds")
	// Run() will return when blobOut is closed.
	s.Run(main, []string{"--stdin", dsName})

	db, blob, err := spec.GetPath(dsName + ".value")
	assert.NoError(err)

	expected := types.NewBlob(bytes.NewBufferString("abcdef"))
	assert.True(expected.Equals(blob))

	meta := db.Head("ds").Get(datas.MetaField).(types.Struct)
	// The meta should only have a "date" field.
	metaDesc := meta.Type().Desc.(types.StructDesc)
	assert.Equal(1, metaDesc.Len())
	assert.NotNil(metaDesc.Field("date"))

	db.Close()
}

func (s *testSuite) TestImportFromFile() {
	assert := s.Assert()

	f, err := ioutil.TempFile("", "TestImportFromFile")
	assert.NoError(err)

	f.Write([]byte("abcdef"))
	f.Close()

	dsName := spec.CreateValueSpecString("ldb", s.LdbDir, "ds")
	s.Run(main, []string{f.Name(), dsName})

	db, blob, err := spec.GetPath(dsName + ".value")
	assert.NoError(err)

	expected := types.NewBlob(bytes.NewBufferString("abcdef"))
	assert.True(expected.Equals(blob))

	meta := db.Head("ds").Get(datas.MetaField).(types.Struct)
	metaDesc := meta.Type().Desc.(types.StructDesc)
	assert.Equal(2, metaDesc.Len())
	assert.NotNil(metaDesc.Field("date"))
	assert.Equal(f.Name(), string(meta.Get("file").(types.String)))

	db.Close()
}

// TODO: TestImportFromURL

// Copyright 2016 The go-gclchaineum Authors
// This file is part of go-gclchaineum.
//
// go-gclchaineum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-gclchaineum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-gclchaineum. If not, see <http://www.gnu.org/licenses/>.

// Command bzzhash computes a swarm tree hash.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gclchaineum/go-gclchaineum/cmd/utils"
	"github.com/gclchaineum/go-gclchaineum/swarm/storage"
	"gopkg.in/urfave/cli.v1"
)

var hashCommand = cli.Command{
	Action:             hash,
	CustomHelpTemplate: helpTemplate,
	Name:               "hash",
	Usage:              "print the swarm hash of a file or directory",
	ArgsUsage:          "<file>",
	Description:        "Prints the swarm hash of file or directory",
}

func hash(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		utils.Fatalf("Usage: swarm hash <file name>")
	}
	f, err := os.Open(args[0])
	if err != nil {
		utils.Fatalf("Error opening file " + args[1])
	}
	defer f.Close()

	stat, _ := f.Stat()
	fileStore := storage.NewFileStore(&storage.FakeChunkStore{}, storage.NewFileStoreParams())
	addr, _, err := fileStore.Store(context.TODO(), f, stat.Size(), false)
	if err != nil {
		utils.Fatalf("%v\n", err)
	} else {
		fmt.Printf("%v\n", addr)
	}
}

package ingestion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func sorted(s []string) []string {
	cp := make([]string, len(s))
	copy(cp, s)
	sort.Strings(cp)
	return cp
}

func TestParseGoImports(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "main.go", `package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/brentrockwood/mri/schema"
)

func main() {}
`)

	got, err := ParseImports(context.Background(), path, "go")
	if err != nil {
		t.Fatalf("ParseImports: %v", err)
	}

	want := []string{
		"context",
		"fmt",
		"github.com/brentrockwood/mri/schema",
		"github.com/spf13/cobra",
	}
	if diff := sliceDiff(sorted(got), want); diff != "" {
		t.Errorf("ParseImports mismatch (-got +want):\n%s", diff)
	}
}

func TestParsePythonImports(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "app.py", `
import os
import sys
from collections import OrderedDict
from pathlib import Path
import re, json
`)

	got, err := ParseImports(context.Background(), path, "python")
	if err != nil {
		t.Fatalf("ParseImports: %v", err)
	}

	want := []string{"collections", "json", "os", "pathlib", "re", "sys"}
	if diff := sliceDiff(sorted(got), want); diff != "" {
		t.Errorf("ParseImports mismatch (-got +want):\n%s", diff)
	}
}

func TestParseJSImports(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "index.js", `
import React from 'react';
import { useState } from 'react';
import styles from './styles.css';
const lodash = require('lodash');
import 'side-effect';
`)

	got, err := ParseImports(context.Background(), path, "javascript")
	if err != nil {
		t.Fatalf("ParseImports: %v", err)
	}

	want := []string{"./styles.css", "lodash", "react", "side-effect"}
	if diff := sliceDiff(sorted(got), want); diff != "" {
		t.Errorf("ParseImports mismatch (-got +want):\n%s", diff)
	}
}

func TestParseTSImports(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "server.ts", `
import express from 'express';
import { Request, Response } from 'express';
import type { Config } from './config';
`)

	got, err := ParseImports(context.Background(), path, "typescript")
	if err != nil {
		t.Fatalf("ParseImports: %v", err)
	}

	want := []string{"./config", "express"}
	if diff := sliceDiff(sorted(got), want); diff != "" {
		t.Errorf("ParseImports mismatch (-got +want):\n%s", diff)
	}
}

func TestParseJavaImports(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "Main.java", `
package com.example;

import java.util.List;
import java.util.ArrayList;
import static java.util.Collections.sort;
import com.google.common.collect.ImmutableList;
`)

	got, err := ParseImports(context.Background(), path, "java")
	if err != nil {
		t.Fatalf("ParseImports: %v", err)
	}

	want := []string{
		"com.google.common.collect.ImmutableList",
		"java.util.ArrayList",
		"java.util.Collections.sort",
		"java.util.List",
	}
	if diff := sliceDiff(sorted(got), want); diff != "" {
		t.Errorf("ParseImports mismatch (-got +want):\n%s", diff)
	}
}

func TestParseImportsUnsupportedLanguage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "script.sh", "#!/bin/bash\necho hello\n")

	got, err := ParseImports(context.Background(), path, "shell")
	if err != nil {
		t.Fatalf("ParseImports: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected nil/empty for unsupported language, got %v", got)
	}
}

func TestParseImportsContextCancelled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := writeTempFile(t, dir, "main.go", "package main\n")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ParseImports(ctx, path, "go")
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}

// sliceDiff returns a human-readable diff string if the two slices differ,
// or an empty string if they are equal.
func sliceDiff(got, want []string) string {
	if len(got) != len(want) {
		return fmt.Sprintf("len(got)=%d, len(want)=%d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			return fmt.Sprintf("index %d: got %q, want %q\ngot:  %v\nwant: %v", i, got[i], want[i], got, want)
		}
	}
	return ""
}

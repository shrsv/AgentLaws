package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/athreyac4/agentlaws/pkg/alaws"
)

// resolveBook determines which book a command should operate on:
//
//  1. explicit, if non-empty (a positional <book> arg or --book flag the
//     caller gave directly), wins outright.
//  2. If <flagRoot>/alaws.toml exists, flagRoot itself is the book - the
//     common case of running alaws from inside one.
//  3. Otherwise, every book under flagRoot is discovered. Exactly one
//     match is used automatically. More than one, in an interactive
//     terminal and without --json, prompts for a choice; otherwise (or
//     with zero matches) returns a clear, actionable error instead of a
//     raw filesystem error (docs/PLAN1.md §32).
func resolveBook(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	if _, err := os.Stat(filepath.Join(flagRoot, "alaws.toml")); err == nil {
		return flagRoot, nil
	}

	books, err := alaws.Discover(flagRoot)
	if err != nil {
		return "", err
	}

	switch len(books) {
	case 0:
		return "", &UsageError{Msg: fmt.Sprintf(
			"no lawbook found under %q; pass a book path, or create one with 'alaws books create <path>'", flagRoot)}
	case 1:
		return books[0].Path, nil
	}

	if !flagJSON && isInteractive() {
		labels := make([]string, len(books))
		for i, b := range books {
			labels[i] = fmt.Sprintf("%s  %s", b.Path, bookLabel(b))
		}
		i, err := promptChoice(fmt.Sprintf("Multiple lawbooks found under %q:", flagRoot), labels)
		if err != nil {
			return "", err
		}
		return books[i].Path, nil
	}

	return "", &UsageError{Msg: multiBookMessage(flagRoot, books)}
}

// resolveBookForUI is like resolveBook but never prompts on stdin: `serve`
// and `ui` hand off ambiguity to the browser's own book picker instead of
// blocking on a terminal prompt (docs/PLAN1.md §32). ok is false when no
// single book could be resolved (zero or multiple candidates); the caller
// should start the server without a book pinned in that case.
func resolveBookForUI(explicit string) (book string, ok bool) {
	if explicit != "" {
		return explicit, true
	}
	if _, err := os.Stat(filepath.Join(flagRoot, "alaws.toml")); err == nil {
		return flagRoot, true
	}
	books, err := alaws.Discover(flagRoot)
	if err != nil || len(books) != 1 {
		return "", false
	}
	return books[0].Path, true
}

// resolveBooks is resolveBook's variadic sibling, for commands like
// `compile`/`validate` that can operate on several books at once
// (docs/PLAN1.md §57). Explicit args win outright, same as resolveBook;
// with none given, it resolves to every book discovered under flagRoot
// rather than prompting for just one - "compile everything under here" is
// the sensible default for a command already designed to take many books,
// unlike the single-book commands resolveBook serves.
func resolveBooks(args []string) ([]string, error) {
	if len(args) > 0 {
		return args, nil
	}
	if _, err := os.Stat(filepath.Join(flagRoot, "alaws.toml")); err == nil {
		return []string{flagRoot}, nil
	}
	books, err := alaws.Discover(flagRoot)
	if err != nil {
		return nil, err
	}
	if len(books) == 0 {
		return nil, &UsageError{Msg: fmt.Sprintf(
			"no lawbook found under %q; pass a book path, or create one with 'alaws books create <path>'", flagRoot)}
	}
	paths := make([]string, len(books))
	for i, b := range books {
		paths[i] = b.Path
	}
	return paths, nil
}

func bookLabel(b alaws.BookInfo) string {
	if b.Title == "" {
		return "(untitled)"
	}
	return b.Title
}

func multiBookMessage(root string, books []alaws.BookInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "multiple lawbooks found under %q; pass one explicitly:", root)
	for _, book := range books {
		fmt.Fprintf(&b, "\n  %s  %s", book.Path, bookLabel(book))
	}
	return b.String()
}

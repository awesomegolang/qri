package cmd

import (
	"testing"
)

// Test that deleting an entire dataset works properly with the logbook.
func TestLogAndDeletes(t *testing.T) {
	run := NewTestRunner(t, "test_peer", "qri_test_log_and_deletes")
	defer run.Delete()

	// Save a dataset containing a body.json, no meta, nothing special.
	err := run.ExecCommand("qri save --body=testdata/movies/body_two.json me/log_test")
	if err != nil {
		t.Fatal(err)
	}

	// Log should have exactly one version
	err = run.ExecCommand("qri log me/log_test")
	if err != nil {
		t.Fatal(err)
	}
	output := run.GetCommandOutput()
	expect := `1   Commit:  /ipfs/QmWL3TSsbTARyJKHehG3dg1iCFqRrgikHKtFCGqsc1GD9z
    Date:    Sun Dec 31 20:01:01 EST 2000
    Storage: local
    Size:    79 B

    created dataset from body_two.json

`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri log (-want +got):\n%s", diff)
	}

	// Save anoter dataset version
	err = run.ExecCommand("qri save --body=testdata/movies/body_four.json me/log_test")
	if err != nil {
		t.Fatal(err)
	}

	// Log should have two versions
	err = run.ExecCommand("qri log me/log_test")
	if err != nil {
		t.Fatal(err)
	}
	output = run.GetCommandOutput()
	expect = `1   Commit:  /ipfs/Qmb6dWPCsAm349UmCboVWAwAkYdjDZttzBmcGV5x3GG4CC
    Date:    Sun Dec 31 20:02:01 EST 2000
    Storage: local
    Size:    137 B

    body added row 2 and added row 3
    body:
    	added row 2
    	added row 3

2   Commit:  /ipfs/QmWL3TSsbTARyJKHehG3dg1iCFqRrgikHKtFCGqsc1GD9z
    Date:    Sun Dec 31 20:01:01 EST 2000
    Storage: local
    Size:    79 B

    created dataset from body_two.json

`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri log (-want +got):\n%s", diff)
	}

	// Save anoter dataset version
	err = run.ExecCommand("qri remove --revisions=1 me/log_test")
	if err != nil {
		t.Fatal(err)
	}

	// Log should only have the first version
	err = run.ExecCommand("qri log me/log_test")
	if err != nil {
		t.Fatal(err)
	}
	output = run.GetCommandOutput()
	expect = `1   Commit:  /ipfs/QmWL3TSsbTARyJKHehG3dg1iCFqRrgikHKtFCGqsc1GD9z
    Date:    Sun Dec 31 20:01:01 EST 2000
    Storage: local
    Size:    79 B

    created dataset from body_two.json

`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri log (-want +got):\n%s", diff)
	}

	// TODO(dlong): Get the logbook log, verify that it contains 4 entries (init, 2 saves, delete).
	// Don't render the raw output, just inspect the number of rows, so that we don't couple
	// tightly with the logbook's internal representation.

	// Save anoter dataset version
	err = run.ExecCommand("qri remove --all me/log_test")
	if err != nil {
		t.Fatal(err)
	}

	// Save anoter dataset version
	err = run.ExecCommand("qri save --body=testdata/movies/body_ten.csv me/log_test")
	if err != nil {
		t.Fatal(err)
	}

	// Log should only have the new version
	err = run.ExecCommand("qri log me/log_test")
	if err != nil {
		t.Fatal(err)
	}
	output = run.GetCommandOutput()
	expect = `1   Commit:  /ipfs/QmekAnaitB9AmTNSFb5UyrKNAdwmkHf8chry5HcxSKCW6g
    Date:    Sun Dec 31 20:03:01 EST 2000
    Storage: local
    Size:    224 B

    created dataset from body_ten.csv

`
	if diff := cmpTextLines(expect, output); diff != "" {
		t.Errorf("qri log (-want +got):\n%s", diff)
	}

	// TODO(dlong): Get the logbook, verify that it contains 2 books. The first should end
	// with a deletion, the second should have only two entries (1 init, 1 save).
}

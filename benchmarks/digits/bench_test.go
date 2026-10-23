package main

import "testing"

func TestSyntheticModelVotesZero(t *testing.T) {
	raw := syntheticTXM()
	model, err := parseModel(raw)
	if err != nil {
		t.Fatal(err)
	}
	if model.nSV != digitsSV {
		t.Fatalf("n_sv = %d, want %d", model.nSV, digitsSV)
	}
	if model.gamma != digitsGamma {
		t.Fatalf("gamma = %g, want %g", model.gamma, digitsGamma)
	}
	if len(model.support) != digitsSV*digitsFeatures {
		t.Fatalf("support len = %d", len(model.support))
	}
	if len(model.dual) != 9*digitsSV {
		t.Fatalf("dual len = %d", len(model.dual))
	}
	if len(model.intercept) != digitsPairs {
		t.Fatalf("intercept len = %d", len(model.intercept))
	}
	var sum int64
	for _, n := range model.nSupport {
		sum += n
	}
	if sum != digitsSV {
		t.Fatalf("n_support sum = %d, want %d", sum, digitsSV)
	}
	digit := model.voteDigit(model.scores(zeroImage()))
	if digit != 0 {
		t.Fatalf("digit = %d, want 0", digit)
	}
}

func TestBenchImageMatchesDatasetLabel(t *testing.T) {
	dir := sourceDir()
	_, pixels, err := loadImage(dir)
	if err != nil {
		t.Fatal(err)
	}
	digit, index, err := loadLabel(dir)
	if err != nil {
		t.Fatal(err)
	}
	if index != 1728 || digit != 7 {
		t.Fatalf("label = row %d digit %d, want row 1728 digit 7", index, digit)
	}
	raw, _, err := loadModelBytes(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	model, err := parseModel(raw)
	if err != nil {
		t.Fatal(err)
	}
	if model.nSV != digitsSV {
		t.Fatalf("n_sv = %d, want %d", model.nSV, digitsSV)
	}
	got := model.voteDigit(model.scores(pixels))
	if got != digit {
		t.Fatalf("host vote = %d, dataset label = %d", got, digit)
	}
}

func TestParseLevels(t *testing.T) {
	levels, err := parseLevels("1, 8, 32")
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 3 || levels[0] != 1 || levels[1] != 8 || levels[2] != 32 {
		t.Fatalf("levels = %v", levels)
	}
	if _, err := parseLevels("0"); err == nil {
		t.Fatal("expected concurrency 0 to fail")
	}
}

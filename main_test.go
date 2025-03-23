package main

import (
	"reflect"
	"testing"

	_ "github.com/lib/pq"
)

func Test_generateEmbedding(t *testing.T) {
	type args struct {
		text string
	}
	tests := []struct {
		name    string
		args    args
		wantNot []float32
		wantErr bool
	}{
		{
			name: "simple",
			args: args{
				text: "hello world",
			},
			wantNot: []float32{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateEmbedding(tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateEmbedding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.wantNot) {
				t.Errorf("generateEmbedding() = %v, want %v", got, tt.wantNot)
			}
		})
	}
}

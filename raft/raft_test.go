package raft

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestNewRaftNode_EmptyFile tests initialization with a new empty file
func TestNewRaftNode_EmptyFile(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_empty.dat")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer file.Close()

	// Create node
	id := uint64(1)
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
	node, err := NewRaftNode(file, id, addr)

	// Assertions
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if node == nil {
		t.Fatal("expected node to be created")
	}
	if node.id != id {
		t.Errorf("expected id %d, got %d", id, node.id)
	}
	if node.addr != addr {
		t.Errorf("expected addr %v, got %v", addr, node.addr)
	}
	if node.currentRole != Follower {
		t.Errorf("expected Follower state, got %v", node.currentRole)
	}
	if node.currentTerm != 0 {
		t.Errorf("expected currentTerm 0, got %d", node.currentTerm)
	}
	if node.votedFor != 0 {
		t.Errorf("expected votedFor 0, got %d", node.votedFor)
	}
	if node.commitLength != 0 {
		t.Errorf("expected commitLength 0, got %d", node.commitLength)
	}
	if len(node.log) != 0 {
		t.Errorf("expected empty log, got length %d", len(node.log))
	}
}

// TestNewRaftNode_WithPersistedState tests recovery from persisted state
func TestNewRaftNode_WithPersistedState(t *testing.T) {
	// Create temporary file and write state
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_persisted.dat")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Write persisted state
	expectedTerm := uint64(5)
	expectedVotedFor := uint64(3)
	expectedLog := []Log{
		{Term: 1, Msg: Message{}},
		{Term: 2, Msg: Message{}},
		{Term: 3, Msg: Message{}},
	}
	expectedCommitLength := uint64(2)

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(expectedTerm); err != nil {
		t.Fatalf("failed to encode term: %v", err)
	}
	if err := encoder.Encode(expectedVotedFor); err != nil {
		t.Fatalf("failed to encode votedFor: %v", err)
	}
	if err := encoder.Encode(expectedLog); err != nil {
		t.Fatalf("failed to encode log: %v", err)
	}
	if err := encoder.Encode(expectedCommitLength); err != nil {
		t.Fatalf("failed to encode commitLength: %v", err)
	}
	file.Close()

	// Reopen file for reading
	file, err = os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to reopen file: %v", err)
	}
	defer file.Close()

	// Create node
	id := uint64(2)
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8081}
	node, err := NewRaftNode(file, id, addr)

	// Assertions
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if node.currentTerm != expectedTerm {
		t.Errorf("expected currentTerm %d, got %d", expectedTerm, node.currentTerm)
	}
	if node.votedFor != expectedVotedFor {
		t.Errorf("expected votedFor %d, got %d", expectedVotedFor, node.votedFor)
	}
	if node.commitLength != expectedCommitLength {
		t.Errorf("expected commitLength %d, got %d", expectedCommitLength, node.commitLength)
	}
	if len(node.log) != len(expectedLog) {
		t.Errorf("expected log length %d, got %d", len(expectedLog), len(node.log))
	}
	for i, logEntry := range node.log {
		if logEntry.Term != expectedLog[i].Term {
			t.Errorf("log[%d]: expected term %d, got %d", i, expectedLog[i].Term, logEntry.Term)
		}
	}
}

// TestNewRaftNode_CorruptedFile_MissingTerm tests handling of corrupted files
func TestNewRaftNode_CorruptedFile_MissingTerm(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_corrupted.dat")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Write some garbage data
	file.Write([]byte("corrupted data"))
	file.Close()

	// Reopen for reading
	file, err = os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to reopen file: %v", err)
	}
	defer file.Close()

	id := uint64(1)
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
	_, err = NewRaftNode(file, id, addr)

	if err == nil {
		t.Fatal("expected error for corrupted file, got nil")
	}
}

// TestNewRaftNode_PartialData tests handling of incomplete persisted state
func TestNewRaftNode_PartialData(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_partial.dat")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Write only term and votedFor, missing log and commitLength
	encoder := gob.NewEncoder(file)
	encoder.Encode(uint64(3))
	encoder.Encode(uint64(2))
	file.Close()

	// Reopen for reading
	file, err = os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to reopen file: %v", err)
	}
	defer file.Close()

	id := uint64(1)
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
	_, err = NewRaftNode(file, id, addr)

	if err == nil {
		t.Fatal("expected error for incomplete data, got nil")
	}
}

// TestNewRaftNode_NilFile tests handling of nil file (if applicable)
func TestNewRaftNode_NilFile(t *testing.T) {
	id := uint64(1)
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}

	// This should panic or return error depending on implementation
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil file")
		}
	}()

	NewRaftNode(nil, id, addr)
}

// TestNewRaftNode_ZeroID tests initialization with zero ID
func TestNewRaftNode_ZeroID(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_zero_id.dat")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer file.Close()

	id := uint64(0)
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
	node, err := NewRaftNode(file, id, addr)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// Note: This might be invalid in Raft, consider validation
	if node.id != id {
		t.Errorf("expected id %d, got %d", id, node.id)
	}
}

// TestNewRaftNode_NilAddr tests initialization with nil address
func TestNewRaftNode_NilAddr(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_nil_addr.dat")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer file.Close()

	id := uint64(1)
	node, err := NewRaftNode(file, id, nil)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if node.addr != nil {
		t.Error("expected nil addr to be preserved")
	}
}

// TestNewRaftNode_LargeLog tests recovery with a large log
func TestNewRaftNode_LargeLog(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_large_log.dat")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Create large log
	largeLog := make([]Log, 10000)
	for i := range largeLog {
		largeLog[i] = Log{Term: uint64(i / 100), Msg: Message{}}
	}

	encoder := gob.NewEncoder(file)
	encoder.Encode(uint64(100))
	encoder.Encode(uint64(5))
	encoder.Encode(largeLog)
	encoder.Encode(uint64(9999))
	file.Close()

	// Reopen for reading
	file, err = os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to reopen file: %v", err)
	}
	defer file.Close()

	id := uint64(1)
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
	node, err := NewRaftNode(file, id, addr)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(node.log) != len(largeLog) {
		t.Errorf("expected log length %d, got %d", len(largeLog), len(node.log))
	}
}

// TestNewRaftNode_ConcurrentCreation tests thread safety (if applicable)
func TestNewRaftNode_ConcurrentCreation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple nodes concurrently
	nodes := make([]*RaftNode, 5)
	errors := make([]error, 5)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			filePath := filepath.Join(tmpDir, fmt.Sprintf("test_concurrent_%d.dat", idx))
			file, err := os.Create(filePath)
			if err != nil {
				errors[idx] = err
				return
			}
			defer file.Close()

			id := uint64(idx + 1)
			addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080 + idx}
			nodes[idx], errors[idx] = NewRaftNode(file, id, addr)
		}(i)
	}
	wg.Wait()

	// Check all nodes created successfully
	for i, err := range errors {
		if err != nil {
			t.Errorf("node %d creation failed: %v", i, err)
		}
		if nodes[i] == nil {
			t.Errorf("node %d is nil", i)
		}
	}
}

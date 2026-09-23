package kafkaout

import "testing"

func TestUseBrokers(t *testing.T) {
	prev := seedBrokers
	t.Cleanup(func() { seedBrokers = prev })

	UseBrokers(" kafka:29092 , 127.0.0.1:9092 ")
	got := Brokers()
	if len(got) != 2 || got[0] != "kafka:29092" || got[1] != "127.0.0.1:9092" {
		t.Fatalf("%v", got)
	}
	got[0] = "mutated"
	if Brokers()[0] != "kafka:29092" {
		t.Fatal("Brokers returned its internal slice")
	}

	UseBrokers("   ")
	if Brokers() != nil {
		t.Fatalf("blank %v", Brokers())
	}
}

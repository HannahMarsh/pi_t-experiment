package pi_t

//
//func TestFormOnion(t *testing.T) {
//	privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
//	if err != nil {
//		t.Fatalf("KeyGen() error: %v", err)
//	}
//
//	payload := []byte("secret message")
//	publicKeys := []string{publicKeyPEM, publicKeyPEM}
//	routingPath := []string{"node1", "node2"}
//
//	addr, onion, _, err := FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, -1, "")
//	if err != nil {
//		t.Fatalf("FormOnion() error: %v", err)
//	}
//
//	if addr != "node1" {
//		t.Fatalf("FormOnion() expected address 'node1', got %s", addr)
//	}
//
//	if onion == "" {
//		t.Fatal("FormOnion() returned empty onion")
//	}
//}
//
//func TestPeelOnion(t *testing.T) {
//	privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
//	if err != nil {
//		t.Fatalf("KeyGen() error: %v", err)
//	}
//	privateKeyPEM1, publicKeyPEM1, err := keys.KeyGen()
//	if err != nil {
//		t.Fatalf("KeyGen() error: %v", err)
//	}
//	privateKeyPEM2, publicKeyPEM2, err := keys.KeyGen()
//	if err != nil {
//		t.Fatalf("KeyGen() error: %v", err)
//	}
//
//	// client processing
//
//	payload := []byte("secret message")
//	publicKeys := []string{publicKeyPEM1, publicKeyPEM2}
//	routingPath := []string{"node1", "node2"}
//
//	destination, onion, _, err := FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, -1, "")
//	if err != nil {
//		t.Fatalf("FormOnion() error: %v", err)
//	}
//
//	if destination != "node1" {
//		t.Fatalf("PeelOnion() expected destination to be 'node1', got %s", destination)
//	}
//
//	// first hop processing
//
//	peeled, bruises, _, _, err := PeelOnion(onion, privateKeyPEM1)
//	if err != nil {
//		t.Fatalf("PeelOnion() error: %v", err)
//	}
//
//	if bruises != 0 {
//		t.Fatalf("PeelOnion() expected bruises 0, got %d", bruises)
//	}
//	if peeled.NextHop != "node2" {
//		t.Fatalf("PeelOnion() expected next hop 'node1', got %s", peeled.NextHop)
//	}
//
//	headerAdded, err := AddHeader(peeled, 1, privateKeyPEM1, publicKeyPEM1)
//
//	// second hop processing
//
//	peeled2, bruises2, _, _, err := PeelOnion(headerAdded, privateKeyPEM2)
//	if err != nil {
//		t.Fatalf("PeelOnion() error: %v", err)
//	}
//	if bruises2 != 1 {
//		t.Fatalf("PeelOnion() expected bruises 1, got %d", bruises2)
//	}
//
//	if peeled2.NextHop != "" {
//		t.Fatalf("PeelOnion() expected next hop '', got %s", peeled2.NextHop)
//	}
//
//	if peeled2.Payload != string(payload) {
//		t.Fatalf("PeelOnion() expected payload %s, got %s", string(payload), peeled.Payload)
//	}
//}
//
//func TestPeelOnion2(t *testing.T) {
//
//	pl.SetUpLogrusAndSlog("debug")
//
//	var err error
//
//	numNodes := 10
//
//	type node struct {
//		privateKeyPEM string
//		publicKeyPEM  string
//		address       string
//	}
//
//	nodes := make([]node, numNodes)
//
//	for i := 0; i < numNodes; i++ {
//		privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
//		if err != nil {
//			t.Fatalf("KeyGen() error: %v", err)
//		}
//		nodes[i] = node{privateKeyPEM, publicKeyPEM, fmt.Sprintf("node%d", i)}
//	}
//
//	secretMessage := "secret message"
//
//	payload, err := json.Marshal(structs.Message{
//		Msg:  secretMessage,
//		To:   nodes[numNodes-1].address,
//		From: nodes[0].address,
//	})
//	if err != nil {
//		slog.Error("json.Marshal() error", err)
//		t.Fatalf("json.Marshal() error: %v", err)
//	}
//
//	publicKeys := utils.Map(nodes, func(n node) string { return n.publicKeyPEM })
//	routingPath := utils.Map(nodes, func(n node) string { return n.address })
//
//	_, onionStr, _, err := FormOnion(nodes[0].privateKeyPEM, nodes[0].publicKeyPEM, payload, publicKeys[1:], routingPath[1:], -1, nodes[0].address)
//	if err != nil {
//		t.Fatalf("FormOnion() error: %v", err)
//	}
//
//	slog.Info("Done forming onion")
//
//	for i := 1; i < numNodes-1; i++ {
//		slog.Info("Peeling onion", "i", i)
//
//		onion, _, _, _, err := PeelOnion(onionStr, nodes[i].privateKeyPEM)
//		if err != nil {
//			slog.Error("PeelOnion() error", err)
//			t.Fatalf("PeelOnion() error: %v", err)
//		} else {
//			slog.Info("PeelOnion() success", "i", i)
//		}
//		if onion.NextHop != nodes[i+1].address {
//			pl.LogNewError("PeelOnion() expected next hop '%s', got %s", nodes[i+1].address, onion.NextHop)
//			t.Fatalf("PeelOnion() expected next hop '', got %s", onion.NextHop)
//		}
//		if onion.LastHop != nodes[i-1].address {
//			pl.LogNewError("PeelOnion() expected last hop '%s', got %s", nodes[i-1].address, onion.LastHop)
//			t.Fatalf("PeelOnion() expected last hop '', got %s", onion.LastHop)
//		}
//
//		onionStr, err = AddHeader(onion, 1, nodes[i].privateKeyPEM, nodes[i].publicKeyPEM)
//		if err != nil {
//			slog.Error("AddHeader() error", err)
//			t.Fatalf("AddHeader() error: %v", err)
//		}
//	}
//
//	onion, _, _, _, err := PeelOnion(onionStr, nodes[numNodes-1].privateKeyPEM)
//	if err != nil {
//		t.Fatalf("PeelOnion() error: %v", err)
//	}
//	if onion.NextHop != "" {
//		t.Fatalf("PeelOnion() expected next hop '', got %s", onion.NextHop)
//	}
//
//	var Msg structs.Message
//	err = json.Unmarshal([]byte(onion.Payload), &Msg)
//	if err != nil {
//		slog.Error("json.Unmarshal() error", err)
//		t.Fatalf("json.Unmarshal() error: %v", err)
//	}
//	if Msg.Msg != secretMessage {
//		t.Fatalf("PeelOnion() expected payload %s, got %s", string(payload), onion.Payload)
//	}
//	if Msg.To != nodes[numNodes-1].address {
//		t.Fatalf("PeelOnion() expected to address %s, got %s", nodes[numNodes-1].address, Msg.To)
//	}
//	if Msg.From != nodes[0].address {
//		t.Fatalf("PeelOnion() expected from address %s, got %s", nodes[0].address, Msg.From)
//	}
//}
//
//func TestNonceVerification(t *testing.T) {
//	for i := 0; i < 100; i++ {
//		privateKeyPEM, publicKeyPEM, err := keys.KeyGen()
//		if err != nil {
//			t.Fatalf("KeyGen() error: %v", err)
//		}
//		privateKeyPEM1, publicKeyPEM1, err := keys.KeyGen()
//		if err != nil {
//			t.Fatalf("KeyGen() error: %v", err)
//		}
//		privateKeyPEM2, publicKeyPEM2, err := keys.KeyGen()
//		if err != nil {
//			t.Fatalf("KeyGen() error: %v", err)
//		}
//
//		// Client processing
//		payload := []byte("secret message")
//		publicKeys := []string{publicKeyPEM1, publicKeyPEM2}
//		routingPath := []string{"node1", "node2"}
//
//		destination, onion, _, err := FormOnion(privateKeyPEM, publicKeyPEM, payload, publicKeys, routingPath, -1, "")
//		if err != nil {
//			t.Fatalf("FormOnion() error: %v", err)
//		}
//
//		if destination != "node1" {
//			t.Fatalf("PeelOnion() expected destination to be 'node1', got %s", destination)
//		}
//
//		// First hop processing with nonce verification
//		peeled, bruises, nonceVerification, _, err := PeelOnion(onion, privateKeyPEM1)
//		if err != nil {
//			t.Fatalf("PeelOnion() error: %v", err)
//		}
//
//		if bruises != 0 {
//			t.Fatalf("PeelOnion() expected bruises 0, got %d", bruises)
//		}
//		if peeled.NextHop != "node2" {
//			t.Fatalf("PeelOnion() expected next hop 'node2', got %s", peeled.NextHop)
//		}
//
//		// Check nonce verification
//		if !nonceVerification {
//			t.Fatalf("PeelOnion() nonce verification failed")
//		}
//
//		headerAdded, err := AddHeader(peeled, 1, privateKeyPEM1, publicKeyPEM1)
//		if err != nil {
//			t.Fatalf("AddHeader() error: %v", err)
//		}
//
//		// Second hop processing with nonce verification
//		peeled2, bruises2, nonceVerification2, _, err := PeelOnion(headerAdded, privateKeyPEM2)
//		if err != nil {
//			t.Fatalf("PeelOnion() error: %v", err)
//		}
//		if bruises2 != 1 {
//			t.Fatalf("PeelOnion() expected bruises 1, got %d", bruises2)
//		}
//
//		if peeled2.NextHop != "" {
//			t.Fatalf("PeelOnion() expected next hop '', got %s", peeled2.NextHop)
//		}
//
//		if peeled2.Payload != string(payload) {
//			t.Fatalf("PeelOnion() expected payload %s, got %s", string(payload), peeled2.Payload)
//		}
//
//		// Check nonce verification for second hop
//		if !nonceVerification2 {
//			t.Fatalf("PeelOnion() nonce verification failed on second hop")
//		}
//	}
//}

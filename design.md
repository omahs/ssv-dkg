## Architecture

![flow](./docs/imgs/DKGinit.drawio.png)

#### Basic Flow Description:

1. The initiator creates an initial message, signs it and sends it to all operators (/init)
2. The operators upon receiving initial message check initiator message signature and create their DKG identity:

- new DKG secrets created
- if 5 mins pass after the last init message with ID [24]byte and new init message with the same ID is incoming the DKG instance is recreated
- `Exchange` signed message containing the DKG identity is created
- operator replies to init message with the created `Exchange` message

3. The initiator collects all responses into one message and verifies signatures
4. The initiator sends back to all operators the combined message (/dkg)
5. Operators receive all exchange messages to start the DKG process, responding back to initiator with a signed dkg deal bundle
6. Initiator packs the deal bundles together and sends them back to all operators (/dkg)
7. Operators process dkg bundles and finish the DKG protocol of creating a shared key. After DKG process is finished each operator has a share of the shared key which can be used for signing
8. Operator using its share of the shared key signs a deposit root, encrypts with the initial RSA key the share and sends it to the initiator
9. Initiator receives all messages from operators with signatures/encrypted shares and prepares the deposit data with a signature and saves it as JSON file
10. Initiator prepares a payload for SSV contract
11. After the deposit is successful and SSV contract transaction is accepted, operators can continue with their duties using their share of the distributed key

The result of successful DKG protocol at operator side:

```go
type Result struct {
	QUAL []Node // list of nodes that successfully ran the protocol
	Key  *DistKeyShare // the share of the node
}
type DistKeyShare struct {
    // Coefficients of the public polynomial holding the public key.
    Commits []kyber.Point
    // Share of the distributed secret which is private information. This will be used to sign. All sigs can be aggregated to create a T-threshold signature
    Share *share.PriShare
}
```

Output of an operator after DKG is finished:

```go
	// Operator ID
	OperatorID uint64
	// Operator RSA pubkey
	PubKeyRSA *rsa.PublicKey
	// RequestID for the DKG instance (not used for signing)
	RequestID [24]byte
	// EncryptedShare standard SSV encrypted shares
	EncryptedShare []byte
	// SharePubKey is the share's BLS pubkey
	SharePubKey []byte
	// ValidatorPubKey the resulting public key corresponding to the shared private key
	ValidatorPubKey []byte
	// Partial Operator Signature of Deposit Data
	DepositPartialSignature []byte
	// SSV owner + nonce signature
	OwnerNoncePartialSignature []byte
```

## DKG protocol description

#### Exchange message creation DKG protocol:

1. Upon receiving init message from initiator, operator creates (if not exists for init msg ID[24]byte) a kyber-bls12381 instance consisting of

- randomly generated scalar
- corresponding point in elliptic curve group G1 (384 bit)

2. Creates a signed with exchange message consisting of ID[24]byte and point bits

#### DKG protocol steps at operator after receiving all exchange messages from the initiator

1. Generation of DKG nodes:

- operator ID uint64;
- operators G1 point;

2. Creation of a time phaser
3. DKG time phaser starts DealPhase

- computes a private share for each of the operators ids
- encrypts with a corresponding to the operator BLS public key created at exchange step
- pack all deals together and signs

4. Deal bundle is created and sent back to the initiator

### DKG protocol steps at operator after receiving all deal messages from the initiator:

1. Creates the public polynomial from received bundle
2. For each deal decrypts a deal share
3. Checks if share is valid w.r.t. public commitment
4. Forms a response bundle

Initial message fields:

```go
 ID [16]byte //   random UUID
 // Operators involved in the DKG
 Operators []byte  // [ ID:pubkey ] uint8 ID 1byte + RSA pub key
 // T is the threshold for signing
 T uint64
 // WithdrawalCredentials for deposit data
 WithdrawalCredentials []byte
 // Fork ethereum fork for signing
 Fork [4]byte
  // Owner address
 Owner [20]byte
 // Nonce
 Nonce int
```

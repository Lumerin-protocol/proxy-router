package encryption

import "testing"

func TestDecryption1(t *testing.T) {
	expectedText := "127.0.0.1|joshkeanisawesome|p@$$w0rd|8008"
	keyString := "b0585373d39ada79aba03b609b8687b1deec3d77a67f69bd4982dd0f9496c077"
	ct := "04e6cd6fc74372cb8277788cfa60fa78eef1ab26781a54121581cd6f5f6082fc1e66537e10e650855d38fcd483a8b8f2656a8fc89d2f756044c469e381db43e8603a679d5e5ce3a77726b7d25b1918f86f5c58f77dd1a27b2640b375b51c13469498485b1e998314365bcf3559933ccaf655773b49072764cccec634c717abdaeed4d804438c6f9b978e9535db85b15c02163914efc93e24aecd"
	decryptedText := DecryptData(ct, keyString)
	if expectedText != decryptedText {
		t.Errorf("did not decrypt successfully")
	}
}

func TestDecryption2(t *testing.T) {
	expectedText := "127.0.0.1|joshkeanisawesome|p@$$w0rd|8008"
	keyString := "a2d166249cc0d707ed837ea7008dbccdbbc3afbd0de556dec6eb87c3f59721ac"
	ct := "04ecf01cb642fc9bbb461996078f5135e0a315360bb85873d9c3653f15875661d331ef88d8f2fe79745a16990c682c57aca1a73f0de0bab2cc76f3cf959dd1717eee170b71f1a647ec0a92313d4dc3cb44486c8d6134329f0d2dd5695733fc744097463e33de54ca4015cbdf77a14925cd81e5809856afd32c0bb8861164ebe46abea2e8d6117cd534238879de04074f97f131bc6e6ccc26748e"
	decryptedText := DecryptData(ct, keyString)
	if expectedText != decryptedText {
		t.Errorf("did not decrypt successfully")
	}
}

func TestDecryption3(t *testing.T) {
	expectedText := "127.0.0.1|joshkeanisawesome|p@$$w0rd|8008"
	keyString := "148f03eab6ea7e0cf19715bc676a75d525d9403d5da2d0bed1732ab662f18b5f"
	ct := "041eac9a15c77fd3662cc822ee29c76b4475cd8bdba39a82234e270d97bca348d46b087989893d85aabdd8d2cfcf3b2121bdaba9707a2d0e69f8096c19eaa1297d93fecce07bd481750037bb9c7529982d96f074e23549d73a0b0dbb9a57feb12c69e4008bf689483e13a73f3dd6756c032db699e281f529e1d36884696742b003fb9838d27e6a9657a03d84776f8d6ce3ad4514d132f8bf9e15"
	decryptedText := DecryptData(ct, keyString)
	if expectedText != decryptedText {
		t.Errorf("did not decrypt successfully")
	}
}

func TestDecryption4(t *testing.T) {
	expectedText := "127.0.0.1|joshkeanisawesome|p@$$w0rd|8008"
	keyString := "2821d58aef2b065bce10e6026e57493fe900e1ce41e9b6db777e51145c04b4b2"
	ct := "049630c9b54a7aae8ef1b33149198b84809a6f45a34261734670c4cd82eabd11cc6d01e6033dcb3f7a7f61f018cb247a56ee64c4602c534d51ad179115a08d4178edece7d22f19c5dbca2060cd3b80b0a3ee928e28588925f37568661a67d5eeba8a15bba5f03446921a40a07e375d06c180a7fb19a28e43491d8e898bfd0987f7417f029ddd25c7b78320444ae7624cc2c91b4716d4e55606ad"
	decryptedText := DecryptData(ct, keyString)
	if expectedText != decryptedText {
		t.Errorf("did not decrypt successfully")
	}
}

func TestDecryption5(t *testing.T) {
	expectedText := "127.0.0.1|joshkeanisawesome|p@$$w0rd|8008"
	keyString := "a0ab57e97ef329b5baeef113ff639c9a756872efab957cf81963308de341a460"
	ct := "043a510a966c66079f2ec4c1b2ac980689893daea31158835d3fbe2b3b630af2a18980e478255ba655465cd721ab4f7a9cf69fc5883d066dfbca76029fdf583b95f2a7443380da3ca22955909ef8ff8578cefa084b65a7b2e629c4139acbd3f6e6ec58d7b2b493afa5699cb9171d995a5e5dc526000661307bd587b93fcf13df46830458a9e9bbac6d88f49e541108f0cbb7408a72812012dbb8"
	decryptedText := DecryptData(ct, keyString)
	if expectedText != decryptedText {
		t.Errorf("did not decrypt successfully")
	}
}

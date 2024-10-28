package types

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"

	"cosmossdk.io/errors"
	sdkError "github.com/cosmos/cosmos-sdk/types/errors"
)

// ValidateBasic does a sanity check on the provided data.
func (pubKey *DkimPubKey) Validate() error {
	// url pass the pubkey domain
	if _, err := url.Parse(pubKey.Domain); err != nil {
		return errors.Wrap(sdkError.ErrInvalidRequest, err.Error())
	}
	// make sure the public key is base64 encoded
	if _, err := base64.StdEncoding.DecodeString(pubKey.PubKey); err != nil {
		return errors.Wrap(sdkError.ErrInvalidRequest, err.Error())
	}
	return nil
}

func (pub *DkimPubKey) ComputePoseidonHash() error {
	var pp []byte
	b, _ := pem.Decode([]byte(pub.PubKey))
	p := bytes.NewBuffer(pp)
	pem.Encode(p, b)
	k, e := x509.ParsePKCS1PublicKey(p.Bytes())
	if e != nil {
		return e
	}

	// poseidon.Hash([]*big.Int{
	// 	big.NewInt(int64(2042675158572422735167009601580549693)),
	// 	big.NewInt("2318426925121163447366268266877478490"),
	// 	big.NewInt("1147774667595934040844400996565450529"),
	// 	big.NewInt("2585846613753899425173314975383472766"),
	// 	big.NewInt("1729550870628631316824527689749144826"),
	// 	big.NewInt("1409688764733787577291119235590636170"),
	// 	big.NewInt("2653526314989005305308617746718530524"),
	// 	big.NewInt("737602834602272445014721319074990651"),
	// 	big.NewInt("1108223552850320351953361145401433110"),
	// 	big.NewInt("196998911671740026740284042198980922"),
	// 	big.NewInt("1810975214051689602006218559773860466"),
	// 	big.NewInt("1356973725008685867134185890101517745"),
	// 	big.NewInt("1741745429950802523929336578157878155"),
	// 	big.NewInt("322242294656712334589977633789887989"),
	// 	big.NewInt("1317445847036079731092233939335794482"),
	// 	big.NewInt("1737308978482248574701598258817218345"),
	// 	big.NewInt("3883364526267798178367189328134785"),
	// })
	fmt.Println(k)
	return nil
}

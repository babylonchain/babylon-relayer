package bbnrelayer

import (
	"fmt"

	"github.com/babylonchain/babylon-relayer/config"
	"github.com/syndtr/goleveldb/leveldb"
)

// setClientID sets the clientID for the IBC light client of a Cosmos zone
// so that when restarting the relayer, it does not need to create another
// IBC light client again
func (r *Relayer) setClientID(chainName string, clientID string) error {
	dbPath := config.GetDBPath(r.homePath)
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return fmt.Errorf("error opening LevelDB (%s): %w", dbPath, err)
	}
	err = db.Put([]byte(chainName), []byte(clientID), nil)
	db.Close()
	if err != nil {
		return fmt.Errorf("error writing to LevelDB (%s): %w", dbPath, err)
	}

	return nil
}

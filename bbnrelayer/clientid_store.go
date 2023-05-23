package bbnrelayer

import (
	"fmt"

	"github.com/babylonchain/babylon-relayer/config"
	"github.com/syndtr/goleveldb/leveldb"
)

// setClientID sets the clientID for the IBC light client of a Cosmos zone
// so that when restarting the relayer, it does not need to create another
// IBC light client again
// key: chainID
// value: client ID of the given chain on Babylon
func (r *Relayer) setClientID(chainID string, clientID string) error {
	dbPath := config.GetDBPath(r.homePath)
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return fmt.Errorf("error opening LevelDB (%s): %w", dbPath, err)
	}
	err = db.Put([]byte(chainID), []byte(clientID), nil)
	db.Close()
	if err != nil {
		return fmt.Errorf("error writing to LevelDB (%s): %w", dbPath, err)
	}

	return nil
}

func (r *Relayer) getClientID(chainID string) (string, error) {
	dbPath := config.GetDBPath(r.homePath)
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return "", fmt.Errorf("error opening LevelDB (%s): %w", dbPath, err)
	}
	clientID, err := db.Get([]byte(chainID), nil)
	db.Close()

	// distinguish not found and other errors
	if err == leveldb.ErrNotFound {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("error reading LevelDB (%s): %w", dbPath, err)
	}

	return string(clientID), nil
}

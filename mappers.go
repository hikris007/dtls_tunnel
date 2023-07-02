package dtls_tunnel

import "sync"

type Mappers interface {
	Set(string, *ClientMapper)
	Get(string) *ClientMapper
	Exist(string) bool
	Delete(string)
	Range(func(key string, mapper *ClientMapper) bool)
}

type MappersBasedSyncMap struct {
	mappers *sync.Map
}

func NewMappers() Mappers {
	m := &MappersBasedSyncMap{mappers: &sync.Map{}}
	return m
}

func (mappers *MappersBasedSyncMap) Set(key string, clientMapper *ClientMapper) {
	mappers.mappers.Store(key, clientMapper)
}

func (mappers *MappersBasedSyncMap) Get(key string) *ClientMapper {
	cm, isExist := mappers.mappers.Load(key)
	if !isExist {
		return nil
	}

	return cm.(*ClientMapper)
}

func (mappers *MappersBasedSyncMap) Delete(key string) {
	mappers.mappers.Delete(key)
}

func (mappers *MappersBasedSyncMap) Exist(key string) bool {
	_, isExist := mappers.mappers.Load(key)
	return isExist
}

func (mappers *MappersBasedSyncMap) Range(f func(key string, mapper *ClientMapper) bool) {
	withConvert := func(key interface{}, value interface{}) bool {
		return f(key.(string), value.(*ClientMapper))
	}
	mappers.mappers.Range(withConvert)
}

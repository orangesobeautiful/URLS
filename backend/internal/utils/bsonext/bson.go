package bsonext

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ID(id primitive.ObjectID) bson.M {
	return bson.M{"_id": id}
}
func Name(name string) bson.M {
	return bson.M{"name": name}
}

func Set(v any) bson.M {
	return bson.M{"$set": v}
}

func UnSet(v any) bson.M {
	return bson.M{"$unset": v}
}

type IncInfo struct {
	FieldName string
	Val       int64
}
type IncList []IncInfo

func (i *IncList) ADD(name string, val int64) {
	*i = append(*i, IncInfo{FieldName: name, Val: val})
}

func Inc(diffList []IncInfo) bson.M {
	diffBson := make(bson.M)
	for _, diff := range diffList {
		if diff.Val != 0 {
			diffBson[diff.FieldName] = diff.Val
		}
	}

	return bson.M{"$inc": diffBson}
}

func In[T any](sliceV []T) bson.M {
	return bson.M{"$in": sliceV}
}

func Each[T any](sliceV []T) bson.M {
	return bson.M{"$each": sliceV}
}

func AddToSet(v any) bson.M {
	return bson.M{"$addToSet": v}
}

func Push(v any) bson.M {
	return bson.M{"$push": v}
}

func Pull(v any) bson.M {
	return bson.M{"$pull": v}
}

func Project(v any) bson.M {
	return bson.M{"$project": v}
}

func Match(v any) bson.M {
	return bson.M{"$match": v}
}

func Group(v any) bson.M {
	return bson.M{"$group": v}
}

func Reduce(input, initialValue, in any) bson.M {
	return bson.M{"$reduce": bson.M{
		"input":        input,
		"initialValue": initialValue,
		"in":           in,
	}}
}

func SetUnion(v any) bson.M {
	return bson.M{"$setUnion": v}
}

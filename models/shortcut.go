package models

func InsertDocument(database, collection string, document interface{}) error {
	sess, err := GetDBSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	c := sess.DB(database).C(collection)
	if err := c.Insert(document); err != nil {
		return err
	}
	return nil
}

CREATE TABLE Items (
  OwnerID STRING(36) NOT NULL,
  ItemID STRING(36) NOT NULL,
  Quantity INT64,
) PRIMARY KEY (OwnerID, ItemID);

CREATE TABLE ArchivedItems (
  OwnerID STRING(36) NOT NULL,
  ArchiveID STRING(36) NOT NULL,
  Quantity INT64,
) PRIMARY KEY (OwnerID, ArchiveID);

CREATE TABLE Owners (
  OwnerID STRING(36) NOT NULL,
  Name STRING(100),
) PRIMARY KEY (OwnerID);

CREATE VIEW PlainView SQL SECURITY INVOKER AS
  SELECT t.OwnerID, t.ItemID FROM Items AS t;

CREATE VIEW JoinView SQL SECURITY INVOKER AS
  SELECT t.OwnerID, t.ItemID, a.Name
  FROM Items AS t
  JOIN Owners AS a ON a.OwnerID = t.OwnerID;

CREATE VIEW NestedView SQL SECURITY INVOKER AS
  SELECT p.OwnerID, p.ItemID FROM PlainView AS p;

CREATE VIEW UnionKeysView SQL SECURITY INVOKER AS
  SELECT t.OwnerID, t.ItemID FROM Items AS t
  UNION ALL
  SELECT i.OwnerID, i.ArchiveID AS ItemID FROM ArchivedItems AS i;

CREATE VIEW JoinedToSubqueryView SQL SECURITY INVOKER AS
  SELECT k.OwnerID, k.ItemID, d.Quantity
  FROM Items AS t
  RIGHT JOIN (
    SELECT s.OwnerID, s.ItemID FROM Items AS s
    UNION ALL
    SELECT i.OwnerID, i.ArchiveID AS ItemID FROM ArchivedItems AS i
  ) AS k ON k.OwnerID = t.OwnerID AND k.ItemID = t.ItemID
  LEFT JOIN (
    SELECT s.OwnerID, s.ItemID, s.Quantity FROM Items AS s
    UNION ALL
    SELECT i.OwnerID, i.ArchiveID AS ItemID, i.Quantity FROM ArchivedItems AS i
  ) AS d ON d.OwnerID = k.OwnerID AND d.ItemID = k.ItemID;

CREATE VIEW SubqueryFirstView SQL SECURITY INVOKER AS
  SELECT k.OwnerID, k.ArchiveID, a.Name
  FROM (SELECT i.OwnerID, i.ArchiveID FROM ArchivedItems AS i) AS k
  JOIN Owners AS a ON a.OwnerID = k.OwnerID;

CREATE VIEW SubqueryOverViewView SQL SECURITY INVOKER AS
  SELECT k.OwnerID, k.ItemID FROM (SELECT n.OwnerID, n.ItemID FROM NestedView AS n) AS k;

CREATE VIEW OrderedView SQL SECURITY INVOKER AS
  SELECT t.OwnerID, t.ItemID FROM Items AS t ORDER BY t.OwnerID LIMIT 100;

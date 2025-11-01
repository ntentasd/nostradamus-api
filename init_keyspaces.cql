CREATE KEYSPACE IF NOT EXISTS sensors_data WITH replication = {
'class': 'NetworkTopologyStrategy',
'nostradamus-main': '3'
};

CREATE KEYSPACE IF NOT EXISTS sensors_meta WITH replication = {
'class': 'NetworkTopologyStrategy',
'nostradamus-main': '3'
};

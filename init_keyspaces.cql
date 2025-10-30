CREATE KEYSPACE IF NOT EXISTS sensors_data WITH replication = {
'class': 'NetworkTopologyStrategy',
'datacenter1': '2'
};

CREATE KEYSPACE IF NOT EXISTS sensors_meta WITH replication = {
'class': 'NetworkTopologyStrategy',
'datacenter1': '2'
};

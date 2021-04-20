CREATE TABLE users
(
    username character varying(100) NOT NULL,
    status character varying(100) DEFAULT '',
    team character varying(30) DEFAULT '',
    taler integer DEFAULT 0,
    reputation_points integer DEFAULT 0,
    CONSTRAINT users_pkey PRIMARY KEY (username)
);

CREATE TABLE commands
(
    name character varying(50) NOT NULL,
    value character varying(1000) DEFAULT '',
    CONSTRAINT commands_pkey PRIMARY KEY (name)
);

CREATE TABLE automatic_messages
(
    id SERIAL NOT NULL,
    interval integer NOT NULL,
    active boolean NOT NULL DEFAULT true,
    content character varying(500) NOT NULL,
    CONSTRAINT automatic_messages_pkey PRIMARY KEY (id)
);

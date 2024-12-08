DROP TABLE IF EXISTS settings;
CREATE TABLE settings (
    name VARCHAR(30) NOT NULL, -- 設定名
    value TEXT NOT NULL,       -- 設定値
    PRIMARY KEY (name)
);

DROP TABLE IF EXISTS chair_models;
CREATE TABLE chair_models (
    name VARCHAR(50) NOT NULL, -- 椅子モデル名
    speed INTEGER NOT NULL,    -- 移動速度
    PRIMARY KEY (name)
);

DROP TABLE IF EXISTS chairs;
CREATE TABLE chairs (
    id TEXT NOT NULL,                   -- 椅子ID
    owner_id TEXT NOT NULL,             -- オーナーID
    name VARCHAR(30) NOT NULL,          -- 椅子の名前
    model TEXT NOT NULL,                -- 椅子のモデル
    is_active INT NOT NULL,         -- 配椅子受付中かどうか
    access_token VARCHAR(255) NOT NULL, -- アクセストークン
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL, -- 登録日時
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (id)
);

DROP TABLE IF EXISTS chair_locations;
CREATE TABLE chair_locations (
    id TEXT NOT NULL,                   -- 主キー
    chair_id TEXT NOT NULL,             -- 椅子ID
    latitude INTEGER NOT NULL,          -- 経度
    longitude INTEGER NOT NULL,         -- 緯度
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL, -- 登録日時
    PRIMARY KEY (id)
);

DROP TABLE IF EXISTS users;
CREATE TABLE users (
    id TEXT NOT NULL,                   -- ユーザーID
    username VARCHAR(30) NOT NULL UNIQUE, -- ユーザー名
    firstname VARCHAR(30) NOT NULL,     -- 本名(名前)
    lastname VARCHAR(30) NOT NULL,      -- 本名(名字)
    date_of_birth DATE NOT NULL,        -- 生年月日
    access_token VARCHAR(255) NOT NULL UNIQUE, -- アクセストークン
    invitation_code VARCHAR(30) NOT NULL UNIQUE, -- 招待トークン
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL, -- 登録日時
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (id)
);

DROP TABLE IF EXISTS payment_tokens;
CREATE TABLE payment_tokens (
    user_id TEXT NOT NULL,              -- ユーザーID
    token VARCHAR(255) NOT NULL,        -- 決済トークン
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL, -- 登録日時
    PRIMARY KEY (user_id)
);

DROP TABLE IF EXISTS rides;
CREATE TABLE rides (
    id TEXT NOT NULL,                   -- ライドID
    user_id TEXT NOT NULL,              -- ユーザーID
    chair_id TEXT,                      -- 割り当てられた椅子ID
    pickup_latitude INTEGER NOT NULL,   -- 配車位置(経度)
    pickup_longitude INTEGER NOT NULL,  -- 配車位置(緯度)
    destination_latitude INTEGER NOT NULL, -- 目的地(経度)
    destination_longitude INTEGER NOT NULL, -- 目的地(緯度)
    evaluation INTEGER,                 -- 評価
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL, -- 要求日時
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (id)
);

DROP TABLE IF EXISTS ride_statuses;
CREATE TABLE ride_statuses (
    id TEXT NOT NULL,                   -- 主キー
    ride_id TEXT NOT NULL,              -- ライドID
    status VARCHAR(20) CHECK (status IN ('MATCHING', 'ENROUTE', 'PICKUP', 'CARRYING', 'ARRIVED', 'COMPLETED')) NOT NULL, -- 状態
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL, -- 状態変更日時
    app_sent_at TIMESTAMP WITH TIME ZONE, -- ユーザーへの状態通知日時
    chair_sent_at TIMESTAMP WITH TIME ZONE, -- 椅子への状態通知日時
    PRIMARY KEY (id)
);

DROP TABLE IF EXISTS owners;
CREATE TABLE owners (
    id TEXT NOT NULL,                   -- オーナーID
    name VARCHAR(30) NOT NULL UNIQUE,   -- オーナー名
    access_token VARCHAR(255) NOT NULL UNIQUE, -- アクセストークン
    chair_register_token VARCHAR(255) NOT NULL UNIQUE, -- 椅子登録トークン
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL, -- 登録日時
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (id)
);

DROP TABLE IF EXISTS coupons;
CREATE TABLE coupons (
    user_id TEXT NOT NULL,              -- 所有しているユーザーのID
    code VARCHAR(255) NOT NULL,         -- クーポンコード
    discount INTEGER NOT NULL,          -- 割引額
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL, -- 付与日時
    used_by TEXT,                       -- クーポンが適用されたライドのID
    PRIMARY KEY (user_id, code)
);
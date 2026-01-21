-- News/Berita table for organizations and clubs
CREATE TABLE IF NOT EXISTS news (
    uuid VARCHAR(36) PRIMARY KEY,
    organization_id VARCHAR(36) NULL,
    club_id VARCHAR(36) NULL,
    title VARCHAR(500) NOT NULL,
    slug VARCHAR(500) UNIQUE,
    excerpt TEXT,
    content LONGTEXT,
    image_url VARCHAR(500),
    category ENUM('event', 'pengumuman', 'prestasi', 'lainnya') DEFAULT 'pengumuman',
    status ENUM('draft', 'published') DEFAULT 'draft',
    views INT DEFAULT 0,
    author_name VARCHAR(255),
    author_id VARCHAR(36),
    meta_title VARCHAR(255),
    meta_description TEXT,
    published_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (organization_id) REFERENCES organizations(uuid) ON DELETE CASCADE,
    FOREIGN KEY (club_id) REFERENCES clubs(uuid) ON DELETE CASCADE,
    
    INDEX idx_news_org (organization_id),
    INDEX idx_news_club (club_id),
    INDEX idx_news_status (status),
    INDEX idx_news_category (category),
    INDEX idx_news_published (published_at)
);

-- Products table for marketplace
CREATE TABLE IF NOT EXISTS products (
    uuid VARCHAR(36) PRIMARY KEY,
    organization_id VARCHAR(36) NULL,
    club_id VARCHAR(36) NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE,
    description TEXT,
    price DECIMAL(12,2) NOT NULL,
    sale_price DECIMAL(12,2) NULL,
    category ENUM('equipment', 'apparel', 'accessories', 'training', 'other') DEFAULT 'other',
    stock INT DEFAULT 0,
    status ENUM('draft', 'active', 'sold_out', 'archived') DEFAULT 'draft',
    image_url VARCHAR(500),
    images JSON,
    specifications JSON,
    views INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (organization_id) REFERENCES organizations(uuid) ON DELETE CASCADE,
    FOREIGN KEY (club_id) REFERENCES clubs(uuid) ON DELETE CASCADE,
    
    INDEX idx_products_org (organization_id),
    INDEX idx_products_club (club_id),
    INDEX idx_products_status (status),
    INDEX idx_products_category (category)
);

-- Club invitations table
CREATE TABLE IF NOT EXISTS club_invitations (
    uuid VARCHAR(36) PRIMARY KEY,
    club_id VARCHAR(36) NOT NULL,
    email VARCHAR(255) NOT NULL,
    invited_by VARCHAR(36) NOT NULL,
    status ENUM('pending', 'accepted', 'expired', 'cancelled') DEFAULT 'pending',
    token VARCHAR(100) UNIQUE,
    message TEXT,
    expires_at TIMESTAMP,
    accepted_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (club_id) REFERENCES clubs(uuid) ON DELETE CASCADE,
    
    INDEX idx_invitations_club (club_id),
    INDEX idx_invitations_email (email),
    INDEX idx_invitations_token (token),
    INDEX idx_invitations_status (status)
);

-- Update usernames for all archers with realistic Indonesian names
-- This script generates usernames from full_name in format: firstname.lastname

-- Step 1: Update usernames for archers with 2+ words in full_name
-- Format: firstname.lastname (lowercase)
UPDATE archers a1
INNER JOIN (
    SELECT 
        uuid,
        LOWER(CONCAT(
            TRIM(SUBSTRING_INDEX(full_name, ' ', 1)),
            '.',
            TRIM(SUBSTRING_INDEX(SUBSTRING_INDEX(full_name, ' ', 2), ' ', -1))
        )) AS new_username
    FROM archers
    WHERE (username IS NULL 
        OR username LIKE '%gen%' 
        OR username LIKE '%archer%' 
        OR username LIKE '%test%' 
        OR username LIKE '%dummy%' 
        OR username REGEXP '^[a-z]+[0-9]+$'
        OR username LIKE 'user%'
        OR username LIKE 'archer%'
        OR username LIKE 'test%')
    AND full_name IS NOT NULL
    AND full_name != ''
    AND CHAR_LENGTH(full_name) - CHAR_LENGTH(REPLACE(full_name, ' ', '')) >= 1
) a2 ON a1.uuid = a2.uuid
SET a1.username = a2.new_username
WHERE NOT EXISTS (
    SELECT 1 FROM archers a3 
    WHERE a3.username = a2.new_username 
    AND a3.uuid != a1.uuid
);

-- Step 2: Handle remaining duplicates by appending a short unique suffix
UPDATE archers a1
SET a1.username = CONCAT(
    LOWER(TRIM(SUBSTRING_INDEX(full_name, ' ', 1))),
    '.',
    LOWER(TRIM(SUBSTRING_INDEX(SUBSTRING_INDEX(full_name, ' ', 2), ' ', -1))),
    SUBSTRING(REPLACE(uuid, '-', ''), 1, 4)
)
WHERE (username IS NULL 
    OR username LIKE '%gen%' 
    OR username LIKE '%archer%' 
    OR username LIKE '%test%' 
    OR username LIKE '%dummy%' 
    OR username REGEXP '^[a-z]+[0-9]+$'
    OR username LIKE 'user%'
    OR username LIKE 'archer%'
    OR username LIKE 'test%')
AND full_name IS NOT NULL
AND full_name != ''
AND CHAR_LENGTH(full_name) - CHAR_LENGTH(REPLACE(full_name, ' ', '')) >= 1
AND EXISTS (
    SELECT 1 FROM archers a2 
    WHERE a2.username = LOWER(CONCAT(
        TRIM(SUBSTRING_INDEX(a1.full_name, ' ', 1)),
        '.',
        TRIM(SUBSTRING_INDEX(SUBSTRING_INDEX(a1.full_name, ' ', 2), ' ', -1))
    ))
    AND a2.uuid != a1.uuid
);

-- Step 3: Handle single word names
UPDATE archers a1
INNER JOIN (
    SELECT 
        uuid,
        LOWER(TRIM(SUBSTRING_INDEX(full_name, ' ', 1))) AS new_username
    FROM archers
    WHERE (username IS NULL 
        OR username LIKE '%gen%' 
        OR username LIKE '%archer%' 
        OR username LIKE '%test%' 
        OR username LIKE '%dummy%' 
        OR username REGEXP '^[a-z]+[0-9]+$'
        OR username LIKE 'user%'
        OR username LIKE 'archer%'
        OR username LIKE 'test%')
    AND full_name IS NOT NULL
    AND full_name != ''
    AND (CHAR_LENGTH(full_name) - CHAR_LENGTH(REPLACE(full_name, ' ', '')) = 0
         OR CHAR_LENGTH(TRIM(SUBSTRING_INDEX(full_name, ' ', 1))) = CHAR_LENGTH(TRIM(full_name)))
) a2 ON a1.uuid = a2.uuid
SET a1.username = a2.new_username
WHERE NOT EXISTS (
    SELECT 1 FROM archers a3 
    WHERE a3.username = a2.new_username 
    AND a3.uuid != a1.uuid
);

-- Step 4: Handle single word duplicates
UPDATE archers a1
SET a1.username = CONCAT(
    LOWER(TRIM(SUBSTRING_INDEX(full_name, ' ', 1))),
    SUBSTRING(REPLACE(uuid, '-', ''), 1, 4)
)
WHERE (username IS NULL 
    OR username LIKE '%gen%' 
    OR username LIKE '%archer%' 
    OR username LIKE '%test%' 
    OR username LIKE '%dummy%' 
    OR username REGEXP '^[a-z]+[0-9]+$'
    OR username LIKE 'user%'
    OR username LIKE 'archer%'
    OR username LIKE 'test%')
AND full_name IS NOT NULL
AND full_name != ''
AND (CHAR_LENGTH(full_name) - CHAR_LENGTH(REPLACE(full_name, ' ', '')) = 0
     OR CHAR_LENGTH(TRIM(SUBSTRING_INDEX(full_name, ' ', 1))) = CHAR_LENGTH(TRIM(full_name)))
AND EXISTS (
    SELECT 1 FROM archers a2 
    WHERE a2.username = LOWER(TRIM(SUBSTRING_INDEX(a1.full_name, ' ', 1)))
    AND a2.uuid != a1.uuid
);

-- Step 5: Final cleanup - handle any remaining weird usernames
-- For any that still have weird patterns, use full_name with uuid suffix
UPDATE archers
SET username = CONCAT(
    LOWER(REPLACE(REPLACE(REPLACE(TRIM(full_name), ' ', '.'), '-', ''), '_', '')),
    SUBSTRING(REPLACE(uuid, '-', ''), 1, 6)
)
WHERE (username IS NULL 
    OR username LIKE '%gen%' 
    OR username LIKE '%archer%' 
    OR username LIKE '%test%' 
    OR username LIKE '%dummy%' 
    OR username REGEXP '^[a-z]+[0-9]+$'
    OR username LIKE 'user%'
    OR username LIKE 'archer%'
    OR username LIKE 'test%')
AND full_name IS NOT NULL
AND full_name != '';

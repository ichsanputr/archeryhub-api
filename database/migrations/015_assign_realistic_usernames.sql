-- Assign realistic Indonesian usernames to all archers with generic names
-- Using common Indonesian first and last names

-- Common Indonesian first names
SET @first_names = 'budi,andi,eko,agus,dwi,rian,arif,bayu,dedi,heru,indra,joko,krisna,lukman,maman,nur,okta,prasetyo,rama,surya,taufik,udin,verdi,wawan,yoga,amir,bambang,cahyo,dani,erik,fajar,guntur,hendra,ilham,jaya,kurniawan,leonardo,mario,nugroho,oscar,putra,rizki,satrio,tri,usman,vito,wisnu,yusuf,zainal,adit,bram,candra,dimas,edi,ferdi,galih,hanif,irfan,johan,kris,lutfi,mirza,nando,opik,pratama,radit,sandi,tomi,udin,vino,wahyu,yoga,zaki';
-- Common Indonesian last names  
SET @last_names = 'santoso,widodo,prasetyo,kartika,dewi,rahayu,nurhayati,sutrisno,cahyono,sari,wijaya,kurniawan,setiawan,gunawan,maulana,permana,indrawan,putra,ramadhan,hermawan,kusuma,wardana,pratama,handoko,wijayanto,soedarsono,soekarno,soeharto,soedirman,soepomo,soekardjo,soemarno,soetrisno,soewarno,soedarto,soekarso,soemantri,soewandi,soekarno,soedjono,soekamto,soedjatmiko,soedarsono,soekardjo,soemantri,soewandi,soekarno,soedjono,soekamto,soedjatmiko,soedarsono,soekardjo,soemantri,soewandi,soekarno,soedjono,soekamto,soedjatmiko';

-- This approach requires a stored procedure or application-level logic
-- Instead, let's use a simpler approach with UPDATE and CASE/ROW_NUMBER

-- For now, let's update based on a pattern that creates realistic names
-- We'll use a combination approach: assign names based on row order

UPDATE archers a1
INNER JOIN (
    SELECT 
        uuid,
        ROW_NUMBER() OVER (ORDER BY created_at) as rn
    FROM archers
    WHERE username LIKE '%generasi%' 
       OR username LIKE '%archer%' 
       OR full_name LIKE '%Generasi%'
       OR full_name LIKE '%Archer%'
) a2 ON a1.uuid = a2.uuid
SET a1.username = CASE 
    WHEN MOD(a2.rn, 50) = 1 THEN 'budi.santoso'
    WHEN MOD(a2.rn, 50) = 2 THEN 'sari.dewi'
    WHEN MOD(a2.rn, 50) = 3 THEN 'andi.prasetyo'
    WHEN MOD(a2.rn, 50) = 4 THEN 'rina.kartika'
    WHEN MOD(a2.rn, 50) = 5 THEN 'dwi.cahyono'
    WHEN MOD(a2.rn, 50) = 6 THEN 'maya.sari'
    WHEN MOD(a2.rn, 50) = 7 THEN 'eko.widodo'
    WHEN MOD(a2.rn, 50) = 8 THEN 'lina.nurhayati'
    WHEN MOD(a2.rn, 50) = 9 THEN 'agus.sutrisno'
    WHEN MOD(a2.rn, 50) = 10 THEN 'fitri.rahayu'
    WHEN MOD(a2.rn, 50) = 11 THEN 'rian.wijaya'
    WHEN MOD(a2.rn, 50) = 12 THEN 'dina.kusuma'
    WHEN MOD(a2.rn, 50) = 13 THEN 'bambang.setiawan'
    WHEN MOD(a2.rn, 50) = 14 THEN 'sinta.wardani'
    WHEN MOD(a2.rn, 50) = 15 THEN 'joko.gunawan'
    WHEN MOD(a2.rn, 50) = 16 THEN 'ratna.permana'
    WHEN MOD(a2.rn, 50) = 17 THEN 'arif.maulana'
    WHEN MOD(a2.rn, 50) = 18 THEN 'dewi.indrawan'
    WHEN MOD(a2.rn, 50) = 19 THEN 'bayu.putra'
    WHEN MOD(a2.rn, 50) = 20 THEN 'nina.ramadhan'
    WHEN MOD(a2.rn, 50) = 21 THEN 'dedi.hermawan'
    WHEN MOD(a2.rn, 50) = 22 THEN 'lisa.kurniawan'
    WHEN MOD(a2.rn, 50) = 23 THEN 'heru.wijayanto'
    WHEN MOD(a2.rn, 50) = 24 THEN 'sari.handoko'
    WHEN MOD(a2.rn, 50) = 25 THEN 'indra.pratama'
    WHEN MOD(a2.rn, 50) = 26 THEN 'maya.soedarsono'
    WHEN MOD(a2.rn, 50) = 27 THEN 'krisna.soekardjo'
    WHEN MOD(a2.rn, 50) = 28 THEN 'lukman.soemantri'
    WHEN MOD(a2.rn, 50) = 29 THEN 'maman.soewandi'
    WHEN MOD(a2.rn, 50) = 30 THEN 'nur.soekarno'
    WHEN MOD(a2.rn, 50) = 31 THEN 'okta.soedjono'
    WHEN MOD(a2.rn, 50) = 32 THEN 'prasetyo.soekamto'
    WHEN MOD(a2.rn, 50) = 33 THEN 'rama.soedjatmiko'
    WHEN MOD(a2.rn, 50) = 34 THEN 'surya.widodo'
    WHEN MOD(a2.rn, 50) = 35 THEN 'taufik.santoso'
    WHEN MOD(a2.rn, 50) = 36 THEN 'udin.dewi'
    WHEN MOD(a2.rn, 50) = 37 THEN 'verdi.prasetyo'
    WHEN MOD(a2.rn, 50) = 38 THEN 'wawan.kartika'
    WHEN MOD(a2.rn, 50) = 39 THEN 'yoga.cahyono'
    WHEN MOD(a2.rn, 50) = 40 THEN 'amir.sari'
    WHEN MOD(a2.rn, 50) = 41 THEN 'bambang.widodo'
    WHEN MOD(a2.rn, 50) = 42 THEN 'cahyo.nurhayati'
    WHEN MOD(a2.rn, 50) = 43 THEN 'dani.sutrisno'
    WHEN MOD(a2.rn, 50) = 44 THEN 'erik.rahayu'
    WHEN MOD(a2.rn, 50) = 45 THEN 'fajar.wijaya'
    WHEN MOD(a2.rn, 50) = 46 THEN 'guntur.kusuma'
    WHEN MOD(a2.rn, 50) = 47 THEN 'hendra.setiawan'
    WHEN MOD(a2.rn, 50) = 48 THEN 'ilham.wardani'
    WHEN MOD(a2.rn, 50) = 49 THEN 'jaya.gunawan'
    ELSE CONCAT('user', SUBSTRING(REPLACE(a2.uuid, '-', ''), 1, 8))
END
WHERE NOT EXISTS (
    SELECT 1 FROM archers a3 
    WHERE a3.username = CASE 
        WHEN MOD(a2.rn, 50) = 1 THEN 'budi.santoso'
        WHEN MOD(a2.rn, 50) = 2 THEN 'sari.dewi'
        -- ... (same pattern)
        ELSE CONCAT('user', SUBSTRING(REPLACE(a2.uuid, '-', ''), 1, 8))
    END
    AND a3.uuid != a1.uuid
);

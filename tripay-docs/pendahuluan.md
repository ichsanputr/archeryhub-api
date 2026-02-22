Selamat datang di halaman Dokumentasi API. Disini telah disediakan penjelasan lengkap mengenai daftar API yang tersedia dan cara akses/penggunaan dari masing-masing API

Channel pembayaran kami terbagi menjadi 2 jenis yaitu Open Payment & Closed Payment.

Open Payment :

Nominal pembayaran tidak ditentukan oleh merchant, pelanggan dapat memasukkan nominal berapapun
1 Kode Bayar/Nomor Virtual Account dapat digunakan berkali-kali
Biaya transaksi hanya dapat dibebankan ke merchant
Closed Payment :

Nominal pembayaran ditentukan oleh merchant
1 Kode Bayar/Nomor Virtual Account hanya dapat digunakan sekali
Biaya transaksi dapat dibebankan ke merchant atau ke pelanggan

CONTOH REQUEST CLOSED PAYMENT

<?php

$apiKey       = 'api_key_anda';
$privateKey   = 'private_key_anda';
$merchantCode = 'kode merchant anda';
$merchantRef  = 'nomor referensi merchant anda';
$amount       = 1000000;

$data = [
    'method'         => 'BRIVA',
    'merchant_ref'   => $merchantRef,
    'amount'         => $amount,
    'customer_name'  => 'Nama Pelanggan',
    'customer_email' => 'emailpelanggan@domain.com',
    'customer_phone' => '081234567890',
    'order_items'    => [
        [
            'sku'         => 'FB-06',
            'name'        => 'Nama Produk 1',
            'price'       => 500000,
            'quantity'    => 1,
            'product_url' => 'https://tokokamu.com/product/nama-produk-1',
            'image_url'   => 'https://tokokamu.com/product/nama-produk-1.jpg',
        ],
        [
            'sku'         => 'FB-07',
            'name'        => 'Nama Produk 2',
            'price'       => 500000,
            'quantity'    => 1,
            'product_url' => 'https://tokokamu.com/product/nama-produk-2',
            'image_url'   => 'https://tokokamu.com/product/nama-produk-2.jpg',
        ]
    ],
    'return_url'   => 'https://domainanda.com/redirect',
    'expired_time' => (time() + (24 * 60 * 60)), // 24 jam
    'signature'    => hash_hmac('sha256', $merchantCode.$merchantRef.$amount, $privateKey)
];

$curl = curl_init();

curl_setopt_array($curl, [
    CURLOPT_FRESH_CONNECT  => true,
    CURLOPT_URL            => 'https://tripay.co.id/api/transaction/create',
    CURLOPT_RETURNTRANSFER => true,
    CURLOPT_HEADER         => false,
    CURLOPT_HTTPHEADER     => ['Authorization: Bearer '.$apiKey],
    CURLOPT_FAILONERROR    => false,
    CURLOPT_POST           => true,
    CURLOPT_POSTFIELDS     => http_build_query($data),
    CURLOPT_IPRESOLVE      => CURL_IPRESOLVE_V4
]);

$response = curl_exec($curl);
$error = curl_error($curl);

curl_close($curl);

echo empty($error) ? $response : $error;

?>

CONTOH REQUEST OPEN PAYMENT

<?php

$apiKey       = 'api_key_anda';
$privateKey   = 'private_key_anda';
$merchantCode = 'T0001';
$merchantRef  = 'INV345678';
$method       = 'BCAVA';

$data = [
    'method'        => $method,
    'merchant_ref'  => $merchantRef,
    'customer_name' => 'Nama Pelanggan',
    'signature'     => hash_hmac('sha256', $merchantCode.$method.$merchantRef, $privateKey)
];

$curl = curl_init();

curl_setopt_array($curl, [
    CURLOPT_FRESH_CONNECT  => true,
    CURLOPT_URL            => "https://tripay.co.id/api/open-payment/create",
    CURLOPT_RETURNTRANSFER => true,
    CURLOPT_HEADER         => false,
    CURLOPT_HTTPHEADER     => ['Authorization: Bearer '.$apiKey],
    CURLOPT_FAILONERROR    => false,
    CURLOPT_POST           => true,
    CURLOPT_POSTFIELDS     => http_build_query($data),
    CURLOPT_IPRESOLVE      => CURL_IPRESOLVE_V4
]);

$response = curl_exec($curl);
$error = curl_error($curl);

curl_close($curl);

echo empty($error) ? $response : $error;

?>
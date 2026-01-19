T32426 = kode merchant
api key = DEV-M5UUd2LqMpoejcUbHIzzoKl3Iq1bMoTt3xmzUDjk
private key  = yZpob-TXdyg-krir0-EeEbz-X5P3B


Untuk melakukan request transaksi, Anda harus membuat signature yang akan divalidasi sistem TriPay untuk memastikan integritas data dan pengirim saat ditransmisikan ke sistem TriPay. Pada permintaan request Transaksi baru, signature ini dibuat dari kombinasi Kode Merchant, Nomor referensi dari sistem merchant, dan nominal transaksi

Ketiga data tersebut di-hash menggunakan jenis algoritma HMAC-SHA256 yang dikunci dengan Private Key Merchant. Berikut adalah contoh pembuatan signature.

PHP
Python
NodeJS
Copy
<?php

$privateKey   = 'ytf6ooi2gmlNPfpchd94jDOk8hRWOu';
$merchantCode = 'T0001';
$merchantRef  = 'INV55567';
$amount       = 1500000;

$signature = hash_hmac('sha256', $merchantCode.$merchantRef.$amount, $privateKey);

// result
// 9f167eba844d1fcb369404e2bda53702e2f78f7aa12e91da6715414e65b8c86a

?>

API ini digunakan untuk membuat transaksi baru atau melakukan generate kode pembayaran

Request
Endpoint
Header
Body
Contoh Request
Method	POST
Sandbox URL	https://tripay.co.id/api-sandbox/transaction/create
Production URL	https://tripay.co.id/api/transaction/create

Response Sukses
Response Gagal
Copy
{
  "success": true,
  "message": "",
  "data": {
    "reference": "T0001000000000000006",
    "merchant_ref": "INV345675",
    "payment_selection_type": "static",
    "payment_method": "BRIVA",
    "payment_name": "BRI Virtual Account",
    "customer_name": "Nama Pelanggan",
    "customer_email": "emailpelanggan@domain.com",
    "customer_phone": "081234567890",
    "callback_url": "https://domainanda.com/callback",
    "return_url": "https://domainanda.com/redirect",
    "amount": 1000000,
    "fee_merchant": 1500,
    "fee_customer": 0,
    "total_fee": 1500,
    "amount_received": 998500,
    "pay_code": "57585748548596587",
    "pay_url": null,
    "checkout_url": "https://tripay.co.id/checkout/T0001000000000000006",
    "status": "UNPAID",
    "expired_time": 1582855837,
    "order_items": [
      {
        "sku": "PRODUK1",
        "name": "Nama Produk 1",
        "price": 500000,
        "quantity": 1,
        "subtotal": 500000,
        "product_url": "https://tokokamu.com/product/nama-produk-1",
        "image_url": "https://tokokamu.com/product/nama-produk-1.jpg"
      },
      {
        "sku": "PRODUK2",
        "name": "Nama Produk 2",
        "price": 500000,
        "quantity": 1,
        "subtotal": 500000,
        "product_url": "https://tokokamu.com/product/nama-produk-2",
        "image_url": "https://tokokamu.com/product/nama-produk-2.jpg"
      }
    ],
    "instructions": [
      {
        "title": "Internet Banking",
        "steps": [
          "Login ke internet banking Bank BRI Anda",
          "Pilih menu <b>Pembayaran</b> lalu klik menu <b>BRIVA</b>",
          "Pilih rekening sumber dan masukkan Kode Bayar (<b>57585748548596587</b>) lalu klik <b>Kirim</b>",
          "Detail transaksi akan ditampilkan, pastikan data sudah sesuai",
          "Masukkan kata sandi ibanking lalu klik <b>Request</b> untuk mengirim m-PIN ke nomor HP Anda",
          "Periksa HP Anda dan masukkan m-PIN yang diterima lalu klik <b>Kirim</b>",
          "Transaksi sukses, simpan bukti transaksi Anda"
        ]
      }
    ],
    "qr_string": null,
    "qr_url": null
  }
}

API ini digunakan untuk membuat transaksi baru atau melakukan generate kode pembayaran

Request
Endpoint
Header
Body
Contoh Request
Key	Value	Keterangan
Authorization	Bearer {api_key}	Ganti {api_key} dengan API Key merchant Anda 


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
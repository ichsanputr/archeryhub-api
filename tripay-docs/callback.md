Callback adalah pengiriman notifikasi transaksi dari server TriPay ke server pengguna dengan cara memanggil URL callback milik pengguna dengan membawa data terkait transaksi. Pada saat pembayaran dari pelanggan diselesaikan, maka sistem TriPay akan memberikan notifikasi yang berisi data transaksi yang kemudian dapat dikelola lebih lanjut oleh sistem pengguna.

Anda harus membuat sebuah file/controller/routing yang didaftarkan dan nantinya akan dipanggil oleh sistem TriPay saat transaksi berubah

Selain dikirim pada saat pembayaran sukses, callback juga akan dikirim ketika terjadi perubahan status transaksi sehingga sistem pengguna dapat mengambil tindakan yang sesuai dengan status pembayaran

Callback diamankan dengan adanya tanda tangan (signature) yang wajib Anda validasi untuk memastikan bahwa callback dikirim dari sistem kami dan data callback tidak berubah sewaktu dikirim. Dibawah ini adalah contoh pembuatan signature yang nantinya harus Anda cocokkan dengan X-Callback-Signature yang terkirim bersama dengan notifikasi transaksi.

 URL untuk callback (seharusnya) tidak dapat diakses secara langsung melalui browser karena memerlukan verifikasi signature. Untuk melakukan testing, gunakan menu Callback Tester di https://tripay.co.id/member/developer/callback-tester (mode Production) atau https://tripay.co.id/simulator/console/callback (mode Sandbox/Development)
 Jika sistem Anda menerapkan proses Whitelist IP agar sistem kami dapat mengakses URL callback Anda, silahkan lakukan whitelist pada IP kami berikut:
- 95.111.200.230 (IPv4)
- 2a04:3543:1000:2310:ac92:4cff:fe87:63f9 (IPv6)

Pembuatan Signature

const crypto = require('crypto');
const express = require('express'); // we use ExpressJS as example here

const privateKey = "private_key_anda"
const app = express();

app.use(express.json());

app.post('/endpoint-callback-anda', function (request, response) {
    var json = request.body;
    var signature = crypto.createHmac("sha256", privateKey)
        .update(JSON.stringify(json))
        .digest('hex');

    console.log(signature);
    console.log(json);
});

app.listen(3000);

Request
Endpoint
Header
Body
Contoh Data
Method	:	POST
URL	:	URL callback yang diatur di halaman Merchant atau pada saat request transaksi


Key	Contoh Nilai	Keterangan
Content-Type	application/json	Data callback dikirim menggunakan format JSON
X-Callback-Signature	85d99ec90d36c93dad61a98928ef63	Signature callback
X-Callback-Event	payment_status	Event callback.
Nilai: "payment_status"

contoh req body

{
    "reference": "T0001000023000XXXXX",
    "merchant_ref": "INV123456",
    "payment_method": "BCA Virtual Account",
    "payment_method_code": "BCAVA",
    "total_amount": 200000,
    "fee_merchant": 2000,
    "fee_customer": 0,
    "total_fee": 2000,
    "amount_received": 198000,
    "is_closed_payment": 1,
    "status": "PAID",
    "paid_at": 1608133017,
    "note": null
}

Response

Ketika sistem Anda berhasil menerima callback dari kami, sistem Anda harus merespon dengan format JSON yang telah ditentukan. Apabila sistem kami tidak menerima respon yang sesuai, maka akan dianggap gagal dan sistem kami akan mencoba mengirimkan ulang callback dengan jeda waktu 2 menit hingga maksimal 3 kali.

response yang diharapkan

{
    "success": true
}


contoh handle callback

<?php

// Include file koneksi database
require('db_connection.php');

// Ambil data JSON
$json = file_get_contents('php://input');

// Ambil callback signature
$callbackSignature = isset($_SERVER['HTTP_X_CALLBACK_SIGNATURE'])
    ? $_SERVER['HTTP_X_CALLBACK_SIGNATURE']
    : '';

// Isi dengan private key anda
$privateKey = 'private_key_anda';

// Generate signature untuk dicocokkan dengan X-Callback-Signature
$signature = hash_hmac('sha256', $json, $privateKey);

// Validasi signature
if ($callbackSignature !== $signature) {
    exit(json_encode([
        'success' => false,
        'message' => 'Invalid signature',
    ]));
}

$data = json_decode($json);

if (JSON_ERROR_NONE !== json_last_error()) {
    exit(json_encode([
        'success' => false,
        'message' => 'Invalid data sent by payment gateway',
    ]));
}

// Hentikan proses jika callback event-nya bukan payment_status
if ('payment_status' !== $_SERVER['HTTP_X_CALLBACK_EVENT']) {
    exit(json_encode([
        'success' => false,
        'message' => 'Unrecognized callback event: ' . $_SERVER['HTTP_X_CALLBACK_EVENT'],
    ]));
}

$invoiceId = $db->real_escape_string($data->merchant_ref);
$tripayReference = $db->real_escape_string($data->reference);
$status = strtoupper((string) $data->status);

if ($data->is_closed_payment === 1) {
    $result = $db->query("SELECT * FROM tbl_invoices WHERE id = '{$invoiceId}' AND tripay_reference = '{$tripayReference}' AND status = 'UNPAID' LIMIT 1");

    if (! $result) {
        exit(json_encode([
            'success' => false,
            'message' => 'Invoice not found or already paid: ' . $invoiceId,
        ]));
    }

    while ($invoice = $result->fetch_object()) {
        switch ($status) {
            // handle status PAID
            case 'PAID':
                if (! $db->query("UPDATE tbl_invoices SET status = 'PAID' WHERE id = {$invoice->id}")) {
                    exit(json_encode([
                        'success' => false,
                        'message' => $db->error,
                    ]));
                }
                break;

            // handle status EXPIRED
            case 'EXPIRED':
                if (! $db->query("UPDATE tbl_invoices SET status = 'EXPIRED' WHERE id = {$invoice->id}")) {
                    exit(json_encode([
                        'success' => false,
                        'message' => $db->error,
                    ]));
                }
                break;

            // handle status FAILED
            case 'FAILED':
                if (! $db->query("UPDATE tbl_invoices SET status = 'FAILED' WHERE id = {$invoice->id}")) {
                    exit(json_encode([
                        'success' => false,
                        'message' => $db->error,
                    ]));
                }
                break;

            default:
                exit(json_encode([
                    'success' => false,
                    'message' => 'Unrecognized payment status',
                ]));
        }

        exit(json_encode(['success' => true]));
    }
}
                
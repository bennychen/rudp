var { Rudp } = require('../bin/rudp');
var assert = require('assert');

var idx = 0;
function DumpAndDestroy(p) {
  var str = idx + ': send ';
  var m = p;
  while (m != null) {
    str += '( ';
    for (var i = 0; i < m.size; i++) {
      str += m.buffer[i].toString(16) + ' ';
    }
    str += ')';
    m = m.next;
  }
  str += '\n';
  console.log(str);
  idx++;
  Rudp.RudpPackage.returnRecursively(p);
}

function DumpRecv(u) {
  var tmp = new Uint8Array(Rudp.Rudp.MaxPackageSize);
  var n = u.recv(tmp);
  var str = '';
  while (n != 0) {
    if (n < 0) {
      str += 'CORRUPT\n';
      break;
    }

    str += 'RECV ';
    for (var i = 0; i < n; i++) {
      str += tmp[i];
      if (i < n - 1) {
        str += ' ';
      }
    }
    str += '\n';

    n = u.recv(tmp);
  }

  if (str) {
    console.log(str);
  }
  return str;
}

function arrayEqual(a1, a2) {
  a1.forEach(function (item, index) {
    if (a2[index] !== item) {
      return false;
    }
  });
  return true;
}

function isTrue(condition, msg) {
  assert.equal(condition, true, msg);
}

function isEmpty(val, msg) {
  assert.equal(val, '', msg);
}

function isNull(val, msg) {
  assert.equal(val, null, msg);
}

describe('rudp tests', function () {
  it('correct path', function () {
    var U = new Rudp.Rudp(1, 5, 128);
    var t1 = new Uint8Array([1, 2, 3, 4]);
    var t2 = new Uint8Array([5, 6, 7, 8]);

    U.send(t1, t1.length);
    U.send(t2, t2.length);
    var p = U.update(null, 0, 1);
    assert.equal(
      p && !p.next,
      true,
      'RUDP::update error, should only send one package.'
    );
    assert.equal(
      arrayEqual(
        p.buffer.subarray(0, p.size),
        new Uint8Array([8, 0, 0, 1, 2, 3, 4, 8, 0, 1, 5, 6, 7, 8])
      ),
      true,
      'RUDP::update error, data package to send is wrong.'
    );
    DumpAndDestroy(p);

    var t3 = new Uint8Array([
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      10,
      11,
      12,
      13,
    ]); // 256 bytes
    var t4 = new Uint8Array([4, 3, 2, 1]);

    p = U.update(null, 0, 1);
    assert.equal(
      p && !p.next,
      true,
      'RUDP::update error, should only send one package.'
    );
    assert.equal(
      arrayEqual(
        p.buffer.subarray(0, p.size),
        new Uint8Array([Rudp.Rudp.TypeHeartbeat])
      ),
      true,
      'RUDP::update error, data package to send is wrong.'
    );
    DumpAndDestroy(p);

    U.send(t3, t3.length);
    U.send(t4, t4.length);

    p = U.update(null, 0, 1);
    isTrue(
      p && p.next && !p.next.next,
      'RUDP::update error, should send two package.'
    );
    isTrue(
      p.size == 260 && p.buffer[3] == 2,
      'RUDP::update error, should send a package with id 2 and size 260.'
    );
    isTrue(
      arrayEqual(
        p.next.buffer.subarray(0, p.next.size),
        new Uint8Array([8, 0, 3, 4, 3, 2, 1])
      ),
      'RUDP::update error, data package to send is wrong.'
    );
    DumpAndDestroy(p);

    var r1 = new Uint8Array([02, 00, 00, 02, 00, 03]);
    p = U.update(r1, r1.length, 1);
    isTrue(p && !p.next, 'RUDP::update error, should only send one package.');
    isTrue(
      arrayEqual(
        p.buffer.subarray(0, p.size),
        new Uint8Array([8, 0, 0, 1, 2, 3, 4, 8, 0, 3, 4, 3, 2, 1])
      ),
      'RUDP::update error, data package to resend is wrong.'
    );
    DumpAndDestroy(p);

    var recvResult = DumpRecv(U);
    isEmpty(recvResult, 'RUDP::Recv error, should receive nothing.');

    var r2 = new Uint8Array([5, 0, 1, 1, 5, 0, 3, 3]);
    p = U.update(r2, r2.Length, 1);

    isTrue(p && !p.next, 'RUDP::update error, should only send one package.');
    isTrue(
      arrayEqual(
        p.buffer.subarray(p.size),
        new Uint8Array([
          Rudp.Rudp.TypeRequest,
          0,
          0,
          Rudp.Rudp.TypeRequest,
          0,
          2,
        ])
      ),
      'RUDP:update error: should send 2 TypeRequest messages.'
    );
    DumpAndDestroy(p);

    recvResult = DumpRecv(U);
    isEmpty(recvResult, 'RUDP::Recv error, should receive nothing.');
    DumpAndDestroy(U.update(r2, r2.length, 0)); // duplicated recv

    var r3 = new Uint8Array([5, 0, 0, 0, 5, 0, 5, 5]);
    p = U.update(r3, r3.length, 0);
    DumpAndDestroy(p);
    isNull(p, 'RUDP::update error, should send 0 package.');

    var r4 = new Uint8Array([5, 0, 6, 6]);
    p = U.update(r4, r4.length, 1);
    isTrue(p && !p.next, 'RUDP::update error, should only send one package.');
    isTrue(
      arrayEqual(
        p.buffer.subarray(p.size),
        new Uint8Array([
          Rudp.Rudp.TypeRequest,
          0,
          2,
          Rudp.Rudp.TypeRequest,
          0,
          4,
        ])
      ),
      'RUDP:update error: should send 2 TypeRequest message.'
    );
    DumpAndDestroy(p);

    var r5 = new Uint8Array([5, 0, 2, 2]);
    p = U.update(r5, r5.length, 1);
    isTrue(p && !p.next, 'RUDP::update error, should only send one package.');
    isTrue(
      arrayEqual(
        p.buffer.subarray(p.size),
        new Uint8Array([Rudp.Rudp.TypeRequest, 0, 4])
      ),
      'RUDP:update error: should send 1 TypeRequest message.'
    );
    DumpAndDestroy(p);

    recvResult = DumpRecv(U);
    assert.equal(
      recvResult,
      'RECV 0\nRECV 1\nRECV 2\nRECV 3\n',
      'RUDP:Recv error: should receive 0~3 messages.'
    );
    DumpAndDestroy(U.update(r2, r2.length, 0)); // duplicated recv
  });

  it('test expiration', function () {
    idx = 0;
    var U = new Rudp.Rudp(1, 5, 128);
    var t1 = new Uint8Array([1, 2, 3, 4]);
    U.send(t1, t1.length);
    DumpAndDestroy(U.update(null, 0, 5));

    var r1 = new Uint8Array([2, 0, 0]);
    var p = U.update(r1, r1.Length, 5);
    isTrue(
      p.buffer &&
        !p.next &&
        arrayEqual(
          p.buffer.subarray(p.Size),
          new Uint8Array([Rudp.Rudp.TypeMissing, 0, 0])
        ),
      'RUDP::Update error, should send only send 1 TypeMissing msg.'
    );
    DumpAndDestroy(p);
  });

  it('test add missing', function () {
    idx = 0;
    var U = new Rudp.Rudp(1, 5, 128);
    var r1 = new Uint8Array([5, 0, 1, 1]);
    DumpAndDestroy(U.update(r1, r1.length, 1));

    var r2 = new Uint8Array([Rudp.Rudp.TypeMissing, 0, 0]);
    var p = U.update(r2, r2.Length, 1);
    isTrue(
      p.buffer &&
        !p.next &&
        arrayEqual(
          p.buffer.subarray(p.size),
          new Uint8Array([Rudp.Rudp.TypeHeartbeat])
        ),
      'RUDP::Update error, should send only send 1 heartbeat msg.'
    );
    DumpAndDestroy(p);
  });

  it('test recv big message', function () {
    var r = new Uint8Array([
      0x81,
      4,
      0,
      0,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      10,
      11,
      12,
      13,
    ]);

    var U = new Rudp.Rudp(1, 5, 128);
    U.update(r, r.length, 1);
    DumpRecv(U);
  });

  it('test send big message', function () {
    idx = 0;
    var U = new Rudp.Rudp(1, 5, 128);

    var t0 = [];
    var bytes = new Uint8Array([
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
      2,
      1,
      1,
      1,
      1,
      1,
      1,
      3,
    ]); // 120 bytes
    bytes.forEach(element => {
      t0.push(element);
    });
    var t1 = t0.slice();
    t1.push(2, 1, 1, 1); // 124 bytes
    var t2 = t1.slice();
    t2.push(4); // 125 bytes

    // send one package that is exactly mtu
    U.send(new Uint8Array(t1), t1.length);
    var p = U.update(null, 0, 1);
    isTrue(
      p && !p.next && p.size == 128,
      'RUDP::Update error, should only send one mtu-size package.'
    );
    DumpAndDestroy(p);

    // send one package slightly larger than mtu
    U.send(new Uint8Array(t2), t2.length);
    p = U.update(null, 0, 1);
    isTrue(
      p && !p.next && p.size == 129,
      'RUDP::Update error, should only send one package.'
    );
    DumpAndDestroy(p);

    // send one tiny package first and then one large one to fill mtu
    U.send(new Uint8Array([9]), 1);
    U.send(new Uint8Array(t0), t0.length);
    p = U.update(null, 0, 1);
    isTrue(
      p && !p.next,
      'RUDP::Update error, should only send one mtu-size package.'
    );
    DumpAndDestroy(p);

    // send one tiny package first and then one large one to slightly outrun mtu
    U.send(new Uint8Array([9, 9]), 2);
    U.send(new Uint8Array(t0), t0.length);
    p = U.update(null, 0, 1);
    isTrue(
      p && p.next && !p.next.next && p.size == 5,
      'RUDP::Update error, should send 2 packages.'
    );
    DumpAndDestroy(p);

    // send one tiny package and
    U.send(new Uint8Array([0]), 1);
    U.send(new Uint8Array(t2), t2.length);
    p = U.update(null, 0, 1);
    DumpAndDestroy(p);

    var r = new Uint8Array([5, 0, 100, 2]);
    // send large package with requests
    p = U.update(r, r.length, 1);
    isTrue(
      !!(p && p.next && p.next.next),
      'RUDP::Update error, should send 3 packages.'
    );
    DumpAndDestroy(p);
  });

  function getUint8Bytes(value) {
    const bytes = new Uint8Array(2);
    bytes[0] = value >> 8;
    bytes[1] = value;
    return bytes;
  }

  it('test big ID', function () {
    idx = 0;
    var U = new Rudp.Rudp(1, 5, 128);

    var tmp = new Uint8Array(Rudp.Rudp.MaxPackageSize);
    for (var i = 0; i < 0xfffe; i++) {
      var t = new Uint8Array([5, 0, 0, 0]);
      var idData = getUint8Bytes(0xffff & i);
      Rudp.RudpHelper.blockCopy(idData, 0, t, 1, idData.length);
      U.update(t, t.length, 1);
      U.recv(tmp);
    }

    var t1 = new Uint8Array([5, 0, 2, 2]);
    var p = U.update(t1, t1.length, 1);

    isTrue(p && !p.next, 'RUDP::Update error, should only send one package.');
    isTrue(
      arrayEqual(
        p.buffer.subarray(p.Size),
        new Uint8Array([02, 0xff, 0xfe, 02, 0xff, 0xff, 02, 00, 00, 02, 00, 01])
      ),
      'RUDP::Update error, should send 1 package with 4 requests.'
    );
    DumpAndDestroy(p);
  });

  it('test message pool', function () {
    idx = 0;
    var U = new Rudp.Rudp(1, 5, 128);

    var r1 = new Uint8Array([5, 0, 0, 0, 5, 0, 1, 1, 5, 0, 2, 2]);
    DumpAndDestroy(U.update(r1, r1.length, 0));

    var n = U.debugGetPoolSize();
    assert.equal(n, 0, 'Pool size should be 0.');

    DumpRecv(U); // this return messages to pool

    n = U.debugGetPoolSize();
    assert.equal(n, 3, 'Pool size should be 3.');

    // this should create messages from pool
    U.send(new Uint8Array([6, 0, 3, 3, 3]), 5);
    U.send(new Uint8Array([6, 0, 4, 4, 4]), 5);
    U.send(new Uint8Array([6, 0, 5, 5, 5]), 5);

    n = U.debugGetPoolSize();
    assert.equal(n, 0, 'Pool size should be 0.');

    U.update(null, 0, 10);
    U.update(null, 0, 10); // expired messages are returned to pool

    n = U.debugGetPoolSize();
    assert.equal(n, 3, 'Pool size should be 3.');

    // this should create messages from pool
    var r3 = new Uint8Array([6, 0, 6, 6, 6, 6, 0, 7, 7, 7, 6, 0, 8, 8, 8]);
    DumpAndDestroy(U.update(r3, r3.length, 1));

    n = U.debugGetPoolSize();
    assert.equal(n, 0, 'Pool size should be 0.');
  });

  it('test recv heartbeat', function () {
    var U = new Rudp.Rudp(1, 5, 128);
    var r = new Uint8Array([0]);
    U.update(r, r.Length, 1);
    DumpRecv(U);
  });

  it('test corrupt', function () {
    var U = new Rudp.Rudp(1, 5, 0);
    var r1 = new Uint8Array([1, 0, 0, 0]);
    U.update(r1, r1.length, 1);
    var str = DumpRecv(U);
    assert.equal(str, 'CORRUPT\n', 'Should get a corrupt signal.');

    var r2 = new Uint8Array([200]);
    U.update(r2, 100, 1);
    str = DumpRecv(U);

    assert.equal(str, 'CORRUPT\n', 'Should get a corrupt signal.');

    var r3 = new Uint8Array([2, 1]);
    U.update(r3, r3.length, 1);
    str = DumpRecv(U);
    assert.equal(str, 'CORRUPT\n', 'Should get a corrupt signal.');

    var r4 = new Uint8Array([5, 1, 1]);
    U.update(r4, r4.length, 1);

    str = DumpRecv(U);
    assert.equal(str, 'CORRUPT\n', 'Should get a corrupt signal.');

    var r5 = new Uint8Array([1]);
    U.send(r5, 100);
  });

  it('test large package', function () {
    var U = new Rudp.Rudp(1, 5, 128);

    var buf = new Uint8Array(Rudp.Rudp.MaxPackageSize - 4);
    U.send(buf, buf.length);
    var p = U.update(null, 0, 1);
    isTrue(
      p && p.size == Rudp.Rudp.MaxPackageSize,
      'RUDP::Update error, should send a normal package.'
    );

    buf = new Uint8Array(Rudp.Rudp.MaxPackageSize - 3);
    var ret = U.send(buf, buf.length);
    assert.equal(ret, 1, 'should send message failed');
    p = U.update(null, 0, 1);
    isTrue(
      p.buffer &&
        !p.next &&
        arrayEqual(
          p.buffer.subarray(p.size),
          new Uint8Array([Rudp.Rudp.TypeHeartbeat])
        ),
      'RUDP::Update error, should send only send 1 heartbeat msg.'
    );
    DumpAndDestroy(p);
  });
});

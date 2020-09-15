//  the algorithm is based on http://blog.codingnow.com/2016/03/reliable_udp.html
//  source c code is at https://github.com/cloudwu/rudp

namespace Rudp {
  export class RudpPackage {
    public next: RudpPackage;
    public buffer: Uint8Array;
    public size: number;

    public constructor() {
      this.next = null;
      this.size = 0;
      this.buffer = new Uint8Array(Rudp.MaxPackageSize);
    }

    public static getPoolSize(): number {
      return this._pool.length;
    }

    public static take(): RudpPackage {
      if (this._pool.length > 0) {
        const p = this._pool[this._pool.length - 1];
        this._pool.splice(this._pool.length - 1, 1);
        return p;
      } else {
        const p = new RudpPackage();
        return p;
      }
    }

    public static returnRecursively(p: RudpPackage) {
      let temp: RudpPackage = p;
      while (temp) {
        const n = temp.next;
        RudpPackage.return(temp);
        temp = n;
      }
    }

    public static return(p: RudpPackage) {
      this._pool.push(p);
      p.next = null;
      p.size = 0;
    }

    private static _pool: RudpPackage[] = [];
  }

  export class Rudp {
    public static readonly MaxPackageSize: number = 512;

    //  provider sends heartbeat to consumer to keep alive
    public static readonly TypeHeartbeat: number = 0;
    //  abnormal corrupted message
    public static readonly TypeCorrupt: number = 1;
    //  consumer requests provider to resend message
    public static readonly TypeRequest: number = 2;
    //  provider tells consumer that some message is missing
    public static readonly TypeMissing: number = 3;
    //  provider sends normal message to consumer
    public static readonly TypeNormal: number = 4;

    public constructor(sendDelay: number, expiredTime: number, mtu: number) {
      this._mtu = mtu;
      if (this._mtu < 128) {
        this._mtu = 128;
      }

      this._tempBuffer = new RudpTempBuffer(this._mtu);
      this._sendDelay = sendDelay;
      this._expiredTime = expiredTime;
      this._sendAgain = [];
      this._sendQueue = new RudpMessageQueue();
      this._recvQueue = new RudpMessageQueue();
      this._sendHistroy = new RudpMessageQueue();
      this._currentSendID = 0;
    }

    //  sends a new package out
    public send(buffer: Uint8Array, sz: number): number {
      if (sz > Rudp.MaxPackageSize - Rudp.TypeNormal) {
        // package size is too large
        return 1;
      }

      if (sz > buffer.length) {
        sz = buffer.length;
      }

      const m = this.createMessage(buffer, 0, sz);
      m.id = this._currentSendID;
      this._currentSendID++;
      m.tick = this._currentTick;
      this._sendQueue.push(m);
      return 0;
    }

    //  receives message and returns the size of the new message
    //  0 = no new message
    //  -1 = corrupted connection
    public recv(buffer: Uint8Array): number {
      if (this._corrupt) {
        this._corrupt = false;
        return -1;
      }

      const m = this._recvQueue.pop(this._nextRecvedID);
      if (!m) {
        return 0;
      }

      this._nextRecvedID++;
      if (m.sz > 0) {
        RudpHelper.blockCopy(m.buffer, 0, buffer, 0, m.sz);
      }

      this.deleteMessage(m);
      return m.sz;
    }

    //  update should be called every frame with the time tick,
    //  or when a new package is coming.
    //  received is the actual udp package we received
    //  sz is the size of the package
    //  the package returned from this function should be sent out
    //  and RudpPackage.delete should be called afterwards, otherwise it will cause memory leak
    public update(
      received: Uint8Array,
      sz: number,
      deltaTick: number
    ): RudpPackage {
      this._currentTick = this._currentTick + deltaTick;
      this.clearOutPackage();
      if (received && sz > received.length) {
        sz = received.length;
      }

      if (received) {
        this.extractPackages(received, sz);
      }

      if (this._currentTick >= this._lastExpiredTick + this._expiredTime) {
        this.clearSendExpired(this._lastExpiredTick);
        this._lastExpiredTick = this._currentTick;
      }

      if (this._currentTick >= this._lastSendTick + this._sendDelay) {
        this._sendPackage = this.genOutPackage();
        this._lastSendTick = this._currentTick;
        return this._sendPackage;
      }

      return null;
    }

    //  For unit test only
    public debugGetPoolSize(): number {
      let n = 0;
      let m = this._messagePool;
      while (m) {
        n++;
        m = m.next;
      }

      return n;
    }

    private clearOutPackage() {
      this._sendPackage = null;
    }

    private createMessage(
      buffer: Uint8Array,
      offset: number,
      sz: number
    ): RudpMessage {
      let msg: RudpMessage = this._messagePool;
      if (msg) {
        this._messagePool = msg.next;
        if (msg.buffer.length < sz) {
          console.error('Message size is too big.');
        }
      } else {
        msg = new RudpMessage();
        msg.buffer = new Uint8Array(Rudp.MaxPackageSize);
      }

      msg.sz = sz;
      if (buffer) {
        RudpHelper.blockCopy(buffer, offset, msg.buffer, 0, sz);
      }

      msg.tick = 0;
      msg.id = 0;
      msg.next = null;
      return msg;
    }

    private deleteMessage(m: RudpMessage) {
      m.next = this._messagePool;
      this._messagePool = m;
    }

    private clearSendExpired(tick: number) {
      let m: RudpMessage = this._sendHistroy.head;
      let last: RudpMessage = null;
      while (m) {
        if (m.tick >= tick) {
          break;
        }

        last = m;
        m = m.next;
      }

      if (last) {
        //  free all the message before tick
        last.next = this._messagePool;
        this._messagePool = this._sendHistroy.head;
      }

      this._sendHistroy.head = m;
      if (!m) {
        this._sendHistroy.tail = null;
      }
    }

    private extractPackages(buffer: Uint8Array, sz: number) {
      let offset: number = 0;
      while (sz > 0) {
        let tag: number = buffer[offset];
        //  if tag is at [128, 32K], tag is 2 bytes
        //  otherwise tag is 1 byte
        if (tag > 127) {
          if (sz <= 1) {
            this._corrupt = true;
            return;
          }

          tag = this.toUInt16(this.getID(buffer, offset) - 0x8000);
          offset += 2;
          sz -= 2;
        } else {
          offset++;
          sz--;
        }

        switch (tag) {
          case Rudp.TypeHeartbeat:
            if (this._sendAgain.length === 0) {
              //  request next package id
              this._sendAgain.push(this._nextRecvedID);
            }
            break;
          case Rudp.TypeCorrupt:
            this._corrupt = true;
            return;
          case Rudp.TypeRequest:
          case Rudp.TypeMissing: {
            //  | tag (1 byte) | id (2 bytes) |
            if (sz < 2) {
              this._corrupt = true;
              return;
            }

            const id: number = this.getID(buffer, offset);
            if (tag === Rudp.TypeRequest) {
              this.addRequest(id);
            } else {
              this.addMissing(id);
            }

            offset += 2;
            sz -= 2;
            break;
          }
          default: {
            //  | tag (1~2 bytes) | id (2 bytes) | data |
            //  data is at least 1 byte, so general msg's tag starts from 1
            const dataLength: number = tag - Rudp.TypeNormal;
            if (sz < dataLength + 2) {
              this._corrupt = true;
              return;
            }

            const id: number = this.getID(buffer, offset);
            offset += 2;
            this.insertMessageToRecvQueue(id, buffer, offset, dataLength);
            offset = offset + dataLength;
            sz = sz - (dataLength + 2);
            break;
          }
        }
      }
    }

    private addRequest(id: number) {
      this._sendAgain.push(id);
    }

    private addMissing(id: number) {
      this.insertMessageToRecvQueue(id, null, 0, -1);
    }

    private insertMessageToRecvQueue(
      id: number,
      buffer: Uint8Array,
      offset: number,
      sz: number
    ) {
      if (this.compareID(id, this._nextRecvedID) < 0) {
        // UnityEngine.Debug.LogWarningFormat(
        // "Failed to insert msg with id {0} ({1}/{2}), probably a duplicated receive.",
        // id, this._nextRecvedID, _currentRecvIDMax);
        return;
      }

      if (
        this.compareID(id, this._currentRecvIDMax) > 0 ||
        this._recvQueue.head == null
      ) {
        const m = this.createMessage(buffer, offset, sz);
        m.id = id;
        this._recvQueue.push(m);
        this._currentRecvIDMax = id;
      } else {
        let m = this._recvQueue.head;
        let last: RudpMessage = null;
        while (true) {
          if (this.compareID(m.id, id) > 0) {
            const tmp = this.createMessage(buffer, offset, sz);
            tmp.id = id;
            tmp.next = m;
            if (!last) {
              this._recvQueue.head = tmp;
            } else {
              last.next = tmp;
            }

            return;
          } else if (m.id === id) {
            //  Duplicated message
            return;
          }

          last = m;
          m = m.next;
          if (!m) {
            console.error('should never be here unless bug.');
            break;
          }
        }
      }
    }

    private genOutPackage(): RudpPackage {
      this._tempBuffer.reset();
      this.requestMissing(this._tempBuffer);
      this.replyRequest(this._tempBuffer);
      this.sendMessage(this._tempBuffer);
      if (!this._tempBuffer.head && this._tempBuffer.sz === 0) {
        this._tempBuffer.buffer[0] = Rudp.TypeHeartbeat;
        this._tempBuffer.sz = 1;
      }

      if (this._tempBuffer.sz > 0) {
        this._tempBuffer.createPackageFromBuffer();
      }

      return this._tempBuffer.head;
    }

    //  consumer requests missing packets
    private requestMissing(tmp: RudpTempBuffer) {
      let id: number = this._nextRecvedID;
      let m = this._recvQueue.head;
      while (m) {
        if (this.compareID(m.id, id) > 0) {
          let i = id;
          while (this.compareID(i, m.id) < 0) {
            this.PackRequest(tmp, i, Rudp.TypeRequest);
            i = this.toUInt16(i + 1);
          }
        }

        id = this.toUInt16(m.id + 1);
        m = m.next;
      }
    }

    //  provider replies missing packets requests from consumer
    private replyRequest(tmp: RudpTempBuffer) {
      let history: RudpMessage = this._sendHistroy.head;
      for (let i: number = 0; i < this._sendAgain.length; i++) {
        const id = this._sendAgain[i];
        while (true) {
          if (!history || this.compareID(id, history.id) < 0) {
            // expired
            this.PackRequest(tmp, id, Rudp.TypeMissing);
            break;
          } else if (id === history.id) {
            this.packMessage(tmp, history);
            break;
          }

          history = history.next;
        }
      }

      this._sendAgain = [];
    }

    private compareID(srcID: number, destID: number): number {
      const src: number = srcID;
      const dest: number = destID;
      if (src - dest > 0x8000 || src - dest < -0x8000) {
        return dest - src;
      } else {
        return src - dest;
      }
    }

    private getID(buffer: Uint8Array, offset: number): number {
      return RudpHelper.getUInt16(buffer, offset);
    }

    private sendMessage(tmp: RudpTempBuffer) {
      let m = this._sendQueue.head;
      while (m) {
        this.packMessage(tmp, m);
        m = m.next;
      }

      if (this._sendQueue.head) {
        if (!this._sendHistroy.tail) {
          this._sendHistroy.head = this._sendQueue.head;
          this._sendHistroy.tail = this._sendQueue.tail;
        } else {
          this._sendHistroy.tail.next = this._sendQueue.head;
          this._sendHistroy.tail = this._sendQueue.tail;
        }

        this._sendQueue.head = null;
        this._sendQueue.tail = null;
      }
    }

    private PackRequest(tmp: RudpTempBuffer, id: number, tag: number) {
      const sz: number = this._mtu - tmp.sz;
      if (sz < 3) {
        tmp.createPackageFromBuffer();
      }

      tmp.sz = tmp.sz + this.fillHeader(tmp.buffer, tmp.sz, tag, id);
    }

    private packMessage(tmp: RudpTempBuffer, m: RudpMessage) {
      if (m.sz > this._mtu - 4) {
        if (tmp.sz > 0) {
          tmp.createPackageFromBuffer();
        }

        //  big package
        const sz: number = 4 + m.sz;
        const p = tmp.createEmptyPackage(sz);
        p.next = null;
        p.size = sz;
        this.fillHeader(p.buffer, 0, m.sz + Rudp.TypeNormal, m.id);
        RudpHelper.blockCopy(m.buffer, 0, p.buffer, 4, m.sz);
        return;
      }

      //  the remaining size is not enough to hold the message
      if (this._mtu - tmp.sz < 4 + m.sz) {
        tmp.createPackageFromBuffer();
      }

      const length: number = this.fillHeader(
        tmp.buffer,
        tmp.sz,
        m.sz + Rudp.TypeNormal,
        m.id
      );
      RudpHelper.blockCopy(m.buffer, 0, tmp.buffer, tmp.sz + length, m.sz);
      tmp.sz = tmp.sz + (length + m.sz);
    }

    private fillHeader(
      buffer: Uint8Array,
      offset: number,
      length: number,
      id: number
    ): number {
      let sz: number;
      if (length < 128) {
        buffer[offset] = length;
        sz = 1;
      } else {
        RudpHelper.blockCopy(
          this.getBytes(this.toUInt16(length + 0x8000)),
          0,
          buffer,
          offset,
          2
        );
        sz = 2;
      }

      RudpHelper.blockCopy(
        this.getBytes(this.toUInt16(id)),
        0,
        buffer,
        offset + sz,
        2
      );
      return sz + 2;
    }

    private getBytes(u16: number): Uint8Array {
      RudpHelper.putUInt16(Rudp._doubleBytes, 0, u16);
      return Rudp._doubleBytes;
    }

    private toUInt16(n: number): number {
      return n & 0xffff;
    }

    private _sendDelay: number;
    //  after how long we should send messages
    private _expiredTime: number;
    //  after how long messages in history should be cleared
    private _mtu: number;
    //  maximum transmission unit size, recommended value 512
    private _sendQueue: RudpMessageQueue;
    private _recvQueue: RudpMessageQueue;
    private _sendHistroy: RudpMessageQueue;

    //  keep message history in case we need to resend
    private _messagePool: RudpMessage;
    private _sendPackage: RudpPackage;
    //  returned by RUDP::Update
    private _sendAgain: number[];
    //  package ids to send again
    private _corrupt: boolean;
    private _currentTick: number = 0;
    private _lastSendTick: number = 0;
    private _lastExpiredTick: number = 0;
    private _currentSendID: number = 0;
    private _nextRecvedID: number = 0;
    private _currentRecvIDMax: number = -1;
    private _tempBuffer: RudpTempBuffer;
    private static _doubleBytes: Uint8Array = new Uint8Array(2);
  }

  export class RudpHelper {
    // big endien get UInt16
    public static getUInt16(buffer: Uint8Array, offset: number): number {
      return buffer[offset + 1] | (buffer[offset] << 8);
    }

    // big endien put UInt16
    public static putUInt16(buffer: Uint8Array, offset: number, value: number) {
      buffer[offset] = value >> 8;
      buffer[offset + 1] = value;
    }

    public static blockCopy(
      src: Uint8Array,
      srcOffset: number,
      dst: Uint8Array,
      dstOffset: number,
      count: number
    ) {
      if (
        dstOffset + count < 0 ||
        dstOffset + count > dst.length ||
        srcOffset + count < 0 ||
        srcOffset + count > src.length
      ) {
        throw new Error('blockCopy::array out of bounds');
      }
      for (let i = 0; i < count; i++) {
        dst[dstOffset + i] = src[srcOffset + i];
      }
    }
  }

  class RudpMessage {
    public next: RudpMessage;
    public buffer: Uint8Array;
    public sz: number;
    public id: number;
    public tick: number;
  }

  class RudpMessageQueue {
    public head: RudpMessage;
    public tail: RudpMessage;

    public push(m: RudpMessage) {
      if (!this.tail) {
        this.head = m;
        this.tail = m;
      } else {
        this.tail.next = m;
        this.tail = m;
      }
    }

    public pop(id: number): RudpMessage {
      if (!this.head) {
        return null;
      }

      const m = this.head;
      if (m.id !== id) {
        return null;
      }

      this.head = m.next;
      m.next = null;
      if (!this.head) {
        this.tail = null;
      }

      return m;
    }
  }

  class RudpTempBuffer {
    public buffer: Uint8Array;
    public sz: number = 0;
    public head: RudpPackage;
    public tail: RudpPackage;

    public constructor(mtu: number) {
      this.buffer = new Uint8Array(mtu);
    }

    public reset() {
      this.head = null;
      this.tail = null;
    }

    public createEmptyPackage(sz: number): RudpPackage {
      const p: RudpPackage = RudpPackage.take();
      p.next = null;
      p.size = this.sz;
      if (!this.tail) {
        this.tail = p;
        this.head = p;
      } else {
        this.tail.next = p;
        this.tail = p;
      }

      return p;
    }

    public createPackageFromBuffer(): RudpPackage {
      const p: RudpPackage = this.createEmptyPackage(this.sz);
      RudpHelper.blockCopy(this.buffer, 0, p.buffer, 0, this.sz);
      this.sz = 0;
      return p;
    }
  }
}

this.Rudp = Rudp;

// the algorithm is based on http://blog.codingnow.com/2016/03/reliable_udp.html
// source c code is at https://github.com/cloudwu/rudp

using System;
using System.Collections.Generic;

public class RudpPackage
{
  public RudpPackage Next;
  public byte[] Buffer;
  public int Size;

  private RudpPackage() { }

  public static int GetPoolSize()
  {
    return _pool.Count;
  }

  public static RudpPackage Take()
  {
    if (_pool.Count > 0)
    {
      return _pool.Pop();
    }
    else
    {
      RudpPackage package = new RudpPackage();
      package.Buffer = new byte[Rudp.MaxPackageSize];
      return package;
    }
  }

  public static void ReturnRecursively(RudpPackage package)
  {
    RudpPackage temp = package;
    while (temp != null)
    {
      var n = temp.Next;
      RudpPackage.Return(temp);
      temp = n;
    }
  }

  public static void Return(RudpPackage package)
  {
    _pool.Push(package);
    package.Next = null;
    package.Size = 0;
  }

  private static Stack<RudpPackage> _pool = new Stack<RudpPackage>();
}

public static class RudpHelper
{
  public static ushort GetUshort(byte[] buffer, int offset)
  {
    return (ushort)ReadBigEndian(buffer, offset, sizeof(ushort));
  }

  public static int PutUshort(byte[] buffer, int offset, ushort value)
  {
    WriteBigEndian(buffer, offset, sizeof(ushort), value);
    return offset + sizeof(ushort);
  }

  public static void WriteBigEndian(byte[] buffer, int offset, int count, ulong data)
  {
    if (BitConverter.IsLittleEndian)
    {
      for (int i = 0; i < count; i++)
      {
        buffer[offset + count - 1 - i] = (byte)(data >> i * 8);
      }
    }
    else
    {
      for (int i = 0; i < count; i++)
      {
        buffer[offset + i] = (byte)(data >> i * 8);
      }
    }
  }

  public static ulong ReadBigEndian(byte[] buffer, int offset, int count)
  {
    ulong r = 0;
    if (BitConverter.IsLittleEndian)
    {
      for (int i = 0; i < count; i++)
      {
        r |= (ulong)buffer[offset + count - 1 - i] << i * 8;
      }
    }
    else
    {
      for (int i = 0; i < count; i++)
      {
        r |= (ulong)buffer[offset + i] << i * 8;
      }
    }
    return r;
  }
}

public class Rudp
{
  public const int MaxPackageSize = 1200;
  public const int TypeHeartbeat = 0;       // provider sends heartbeat to consumer to keep alive
  public const int TypeCorrupt = 1;         // abnormal corrupted message
  public const int TypeRequest = 2;         // consumer requests provider to resend message
  public const int TypeMissing = 3;         // provider tells consumer that some message is missing
  public const int TypeNormal = 4;          // provider sends normal message to consumer

  public Rudp(int sendDelay, int expiredTime, int mtu)
  {
    _mtu = mtu;
    if (_mtu < 128)
    {
      _mtu = 128;
    }
    _tempBuffer = new TempBuffer(_mtu);

    _sendDelay = sendDelay;
    _expiredTime = expiredTime;
    _sendAgain = new List<ushort>();
    _sendQueue = new MessageQueue();
    _recvQueue = new MessageQueue();
    _sendHistroy = new MessageQueue();
  }

  // Send sends a new package out
  public void Send(byte[] buffer, int sz)
  {
    if (sz > MaxPackageSize - TypeNormal)
    {
      System.Console.WriteLine("package size is too large.");
      return;
    }
    if (sz > buffer.Length)
    {
      sz = buffer.Length;
    }

    var m = CreateMessage(buffer, 0, sz);
    m.id = _currentSendID;
    _currentSendID++;
    m.tick = _currentTick;
    _sendQueue.push(m);
  }

  // Recv receives message and returns the size of the new message
  // 0 = no new message
  // -1 = corrupted connection
  public int Recv(byte[] buffer)
  {
    if (_corrupt)
    {
      _corrupt = false;
      return -1;
    }
    var m = _recvQueue.pop(_nextRecvedID);
    if (m == null) return 0;
    _nextRecvedID++;
    if (m.sz > 0)
    {
      Buffer.BlockCopy(m.buffer, 0, buffer, 0, m.sz);
    }
    DeleteMessage(m);
    return m.sz;
  }

  // Update should be called every frame with the time tick,
  // or when a new package is coming.
  // received is the actual udp package we received
  // sz is the size of the package
  // the package returned from this function should be sent out 
  // and RudpPackage.Delete should be called afterwards, otherwise it will cause memory leak
  public RudpPackage Update(byte[] received, int sz, int deltaTick)
  {
    _currentTick += deltaTick;
    ClearOutPackage();
    if (received != null && sz > received.Length)
    {
      sz = received.Length;
    }
    if (received != null)
    {
      ExtractPackages(received, sz);
    }

    if (_currentTick >= _lastExpiredTick + _expiredTime)
    {
      ClearSendExpired(_lastExpiredTick);
      _lastExpiredTick = _currentTick;
    }
    if (_currentTick >= _lastSendTick + _sendDelay)
    {
      _sendPackage = GenOutPackage();
      _lastSendTick = _currentTick;
      return _sendPackage;
    }
    return null;
  }

  // For unit test only
  public int DebugGetPoolSize()
  {
    var n = 0;
    var m = _messagePool;
    while (m != null)
    {
      n++;
      m = m.next;
    }
    return n;
  }

  private void ClearOutPackage()
  {
    _sendPackage = null;
  }

  private Message CreateMessage(byte[] buffer, int offset, int sz)
  {
    Message msg = _messagePool;
    if (msg != null)
    {
      _messagePool = msg.next;
      if (msg.buffer.Length < sz)
      {
        UnityEngine.Debug.LogError("Message size is too big.");
      }
    }
    else
    {
      msg = new Message();
      msg.buffer = new byte[MaxPackageSize];
    }

    msg.sz = sz;
    if (buffer != null)
    {
      Buffer.BlockCopy(buffer, offset, msg.buffer, 0, sz);
    }
    msg.tick = 0;
    msg.id = 0;
    msg.next = null;
    return msg;
  }

  private void DeleteMessage(Message m)
  {
    m.next = _messagePool;
    _messagePool = m;
  }

  private void ClearSendExpired(int tick)
  {
    Message m = _sendHistroy.head;
    Message last = null;
    while (m != null)
    {
      if (m.tick >= tick)
      {
        break;
      }
      last = m;
      m = m.next;
    }

    if (last != null)
    {
      // free all the message before tick
      last.next = _messagePool;
      _messagePool = _sendHistroy.head;
    }
    _sendHistroy.head = m;
    if (m == null)
    {
      _sendHistroy.tail = null;
    }
  }

  private void ExtractPackages(byte[] buffer, int sz)
  {
    int offset = 0;
    while (sz > 0)
    {
      ushort tag = buffer[offset];
      // if tag is at [128, 32K], tag is 2 bytes
      // otherwise tag is 1 byte
      if (tag > 127)
      {
        if (sz <= 1)
        {
          _corrupt = true;
          return;
        }
        tag = (ushort)(GetID(buffer, offset) - 0x8000);
        offset += 2;
        sz -= 2;
      }
      else
      {
        offset += 1;
        sz--;
      }

      switch (tag)
      {
        case TypeHeartbeat:
          if (_sendAgain.Count == 0)
          {
            // request next package id
            _sendAgain.Add(_nextRecvedID);
          }
          break;
        case TypeCorrupt:
          _corrupt = true;
          return;
        case TypeRequest:
        case TypeMissing:
          {
            // | tag (1 byte) | id (2 bytes) |
            if (sz < 2)
            {
              _corrupt = true;
              return;
            }
            ushort id = GetID(buffer, offset);
            if (tag == TypeRequest)
            {
              AddRequest(id);
            }
            else
            {
              AddMissing(id);
            }
            offset += 2;
            sz -= 2;
            break;
          }
        default:
          {
            // | tag (1~2 bytes) | id (2 bytes) | data |
            // data is at least 1 byte, so general msg's tag starts from 1
            int dataLength = tag - TypeNormal;
            if (sz < dataLength + 2)
            {
              _corrupt = true;
              return;
            }
            ushort id = GetID(buffer, offset);
            offset += 2;
            InsertMessageToRecvQueue(id, buffer, offset, dataLength);
            offset += dataLength;
            sz -= dataLength + 2;
            break;
          }
      }
    }
  }

  private void AddRequest(ushort id)
  {
    _sendAgain.Add(id);
  }

  private void AddMissing(ushort id)
  {
    InsertMessageToRecvQueue(id, null, 0, -1);
  }

  private void InsertMessageToRecvQueue(ushort id, byte[] buffer, int offset, int sz)
  {
    if (CompareID(id, _nextRecvedID) < 0)
    {
      //UnityEngine.Debug.LogWarningFormat(
      //"Failed to insert msg with id {0} ({1}/{2}), probably a duplicated receive.", 
      //id, _nextRecvedID, _currentRecvIDMax);
      return;
    }
    if (CompareID(id, _currentRecvIDMax) > 0 || _recvQueue.head == null)
    {
      var m = CreateMessage(buffer, offset, sz);
      m.id = id;
      _recvQueue.push(m);
      _currentRecvIDMax = id;
    }
    else
    {
      var m = _recvQueue.head;
      Message last = null;
      while (true)
      {
        if (CompareID(m.id, id) > 0)
        {
          var tmp = CreateMessage(buffer, offset, sz);
          tmp.id = id;
          tmp.next = m;

          if (last == null)
          {
            _recvQueue.head = tmp;
          }
          else
          {
            last.next = tmp;
          }
          return;
        }
        else if (m.id == id)
        {
          // Duplicated message
          return;
        }
        last = m;
        m = m.next;

        if (m == null)
        {
          System.Console.WriteLine("Error::Should never be here unless bug.");
          break;
        }
      }
    }
  }

  /*
      1. request missing ( lookup RUDP::recvQueue )
      2. reply request ( RUDP::sendAgain )
      3. send message ( RUDP::sendQueue )
      4. send heartbeat
  */
  private RudpPackage GenOutPackage()
  {
    _tempBuffer.Reset();
    RequestMissing(_tempBuffer);
    ReplyRequest(_tempBuffer);
    SendMessage(_tempBuffer);

    if (_tempBuffer.head == null && _tempBuffer.sz == 0)
    {
      _tempBuffer.buffer[0] = TypeHeartbeat;
      _tempBuffer.sz = 1;
    }
    if (_tempBuffer.sz > 0)
    {
      _tempBuffer.CreatePackageFromBuffer();
    }
    return _tempBuffer.head;
  }

  // consumer requests missing packets
  private void RequestMissing(TempBuffer tmp)
  {
    ushort id = _nextRecvedID;
    var m = _recvQueue.head;
    while (m != null)
    {
      if (CompareID(m.id, id) > 0)
      {
        for (ushort i = id; CompareID(i, m.id) < 0; i++)
        {
          PackRequest(tmp, i, TypeRequest);
        }
      }
      id = (ushort)(m.id + 1);
      m = m.next;
    }
  }

  // provider replies missing packets requests from consumer
  private void ReplyRequest(TempBuffer tmp)
  {
    Message history = _sendHistroy.head;
    for (int i = 0; i < _sendAgain.Count; i++)
    {
      var id = _sendAgain[i];
      while (true)
      {
        if (history == null || CompareID(id, history.id) < 0)
        {
          //expired
          PackRequest(tmp, id, TypeMissing);
          break;
        }
        else if (id == history.id)
        {
          PackMessage(tmp, history);
          break;
        }
        history = history.next;
      }
    }
    _sendAgain.Clear();
  }

  private int CompareID(ushort srcID, ushort destID)
  {
    int src = (int)srcID;
    int dest = (int)destID;
    if (src - dest > 0x8000 || src - dest < -0x8000)
    {
      return dest - src;
    }
    else
    {
      return src - dest;
    }
  }

  private ushort GetID(byte[] buffer, int offset)
  {
    return RudpHelper.GetUshort(buffer, offset);
  }

  private void SendMessage(TempBuffer tmp)
  {
    var m = _sendQueue.head;
    while (m != null)
    {
      PackMessage(tmp, m);
      m = m.next;
    }

    if (_sendQueue.head != null)
    {
      if (_sendHistroy.tail == null)
      {
        _sendHistroy.head = _sendQueue.head;
        _sendHistroy.tail = _sendQueue.tail;
      }
      else
      {
        _sendHistroy.tail.next = _sendQueue.head;
        _sendHistroy.tail = _sendQueue.tail;
      }
      _sendQueue.head = null;
      _sendQueue.tail = null;
    }
  }

  private void PackRequest(TempBuffer tmp, ushort id, int tag)
  {
    int sz = _mtu - tmp.sz;
    if (sz < 3)
    {
      tmp.CreatePackageFromBuffer();
    }
    tmp.sz += FillHeader(tmp.buffer, tmp.sz, tag, id);
  }

  private void PackMessage(TempBuffer tmp, Message m)
  {
    if (m.sz > _mtu - 4)
    {
      if (tmp.sz > 0)
      {
        tmp.CreatePackageFromBuffer();
      }
      // big package
      int sz = 4 + m.sz;
      var p = tmp.CreateEmptyPackage(sz);
      p.Next = null;
      p.Size = sz;
      FillHeader(p.Buffer, 0, m.sz + TypeNormal, m.id);
      Buffer.BlockCopy(m.buffer, 0, p.Buffer, 4, m.sz);
      return;
    }
    // the remaining size is not enough to hold the message
    if (_mtu - tmp.sz < 4 + m.sz)
    {
      tmp.CreatePackageFromBuffer();
    }
    int length = FillHeader(tmp.buffer, tmp.sz, m.sz + TypeNormal, m.id);
    Buffer.BlockCopy(m.buffer, 0, tmp.buffer, tmp.sz + length, m.sz);
    tmp.sz += length + m.sz;
  }

  private int FillHeader(byte[] buffer, int offset, int length, ushort id)
  {
    int sz;
    if (length < 128)
    {
      buffer[offset] = (byte)length;
      sz = 1;
    }
    else
    {
      Buffer.BlockCopy(GetBytes(Convert.ToUInt16(length + 0x8000)), 0, buffer, offset, 2);
      sz = 2;
    }

    Buffer.BlockCopy(GetBytes(Convert.ToUInt16(id)), 0, buffer, offset + sz, 2);
    return sz + 2;
  }

  private byte[] GetBytes(ushort u16)
  {
    RudpHelper.PutUshort(_doubleBytes, 0, u16);
    return _doubleBytes;
  }

  private class Message
  {
    public Message next;
    public byte[] buffer;
    public int sz;
    public ushort id;
    public int tick;
  }

  private class MessageQueue
  {
    public Message head;
    public Message tail;

    public void push(Message m)
    {
      if (tail == null)
      {
        head = m;
        tail = m;
      }
      else
      {
        tail.next = m;
        tail = m;
      }
    }

    public Message pop(ushort id)
    {
      if (head == null)
      {
        return null;
      }
      var m = head;
      if (m.id != id)
      {
        return null;
      }
      head = m.next;
      m.next = null;
      if (head == null)
      {
        tail = null;
      }
      return m;
    }
  }

  private class TempBuffer
  {
    public byte[] buffer;
    public int sz;
    public RudpPackage head;
    public RudpPackage tail;

    public TempBuffer(int mtu)
    {
      buffer = new byte[mtu];
    }

    public void Reset()
    {
      head = null;
      tail = null;
    }

    public RudpPackage CreateEmptyPackage(int sz)
    {
      RudpPackage p = RudpPackage.Take();
      p.Next = null;
      p.Size = sz;
      if (tail == null)
      {
        tail = p;
        head = p;
      }
      else
      {
        tail.Next = p;
        tail = p;
      }
      return p;
    }

    public RudpPackage CreatePackageFromBuffer()
    {
      RudpPackage p = CreateEmptyPackage(sz);
      Buffer.BlockCopy(buffer, 0, p.Buffer, 0, sz);
      sz = 0;
      return p;
    }
  }

  private int _sendDelay; // after how long we should send messages
  private int _expiredTime; // after how long messages in history should be cleared

  private int _mtu;          // maximum transmission unit size, recommended value 512
  private MessageQueue _sendQueue;
  private MessageQueue _recvQueue;
  private MessageQueue _sendHistroy; // keep message history in case we need to resend
  private Message _messagePool;

  private RudpPackage _sendPackage; // returned by RUDP::Update

  private List<ushort> _sendAgain;     // package ids to send again

  private bool _corrupt;
  private int _currentTick;
  private int _lastSendTick;
  private int _lastExpiredTick;
  private ushort _currentSendID;
  private ushort _nextRecvedID;
  private ushort _currentRecvIDMax;
  private TempBuffer _tempBuffer;

  private static byte[] _doubleBytes = new byte[2];
}

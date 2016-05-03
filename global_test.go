// Copyright 2014 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package loggo_test

import (
	"os"

	gc "gopkg.in/check.v1"

	"github.com/juju/loggo"
)

type GlobalLoggersSuite struct{}

var _ = gc.Suite(&GlobalLoggersSuite{})

func (*GlobalLoggersSuite) SetUpTest(c *gc.C) {
	loggo.ResetLoggers()
}

func (*GlobalLoggersSuite) TestRootLogger(c *gc.C) {
	var root loggo.Logger

	got := loggo.GetLogger("")

	c.Check(got.Name(), gc.Equals, root.Name())
	c.Check(got.LogLevel(), gc.Equals, root.LogLevel())
}

func (*GlobalLoggersSuite) TestModuleName(c *gc.C) {
	logger := loggo.GetLogger("loggo.testing")

	c.Check(logger.Name(), gc.Equals, "loggo.testing")
}

func (*GlobalLoggersSuite) TestLevel(c *gc.C) {
	logger := loggo.GetLogger("testing")

	level := logger.LogLevel()

	c.Check(level, gc.Equals, loggo.UNSPECIFIED)
}

func (*GlobalLoggersSuite) TestEffectiveLevel(c *gc.C) {
	logger := loggo.GetLogger("testing")

	level := logger.EffectiveLogLevel()

	c.Check(level, gc.Equals, loggo.WARNING)
}

func (*GlobalLoggersSuite) TestLevelsSharedForSameModule(c *gc.C) {
	logger1 := loggo.GetLogger("testing.module")
	logger2 := loggo.GetLogger("testing.module")

	logger1.SetLogLevel(loggo.INFO)
	c.Assert(logger1.IsInfoEnabled(), gc.Equals, true)
	c.Assert(logger2.IsInfoEnabled(), gc.Equals, true)
}

func (*GlobalLoggersSuite) TestModuleLowered(c *gc.C) {
	logger1 := loggo.GetLogger("TESTING.MODULE")
	logger2 := loggo.GetLogger("Testing")

	c.Assert(logger1.Name(), gc.Equals, "testing.module")
	c.Assert(logger2.Name(), gc.Equals, "testing")
}

func (*GlobalLoggersSuite) TestLevelsInherited(c *gc.C) {
	root := loggo.GetLogger("")
	first := loggo.GetLogger("first")
	second := loggo.GetLogger("first.second")

	root.SetLogLevel(loggo.ERROR)
	c.Assert(root.LogLevel(), gc.Equals, loggo.ERROR)
	c.Assert(root.EffectiveLogLevel(), gc.Equals, loggo.ERROR)
	c.Assert(first.LogLevel(), gc.Equals, loggo.UNSPECIFIED)
	c.Assert(first.EffectiveLogLevel(), gc.Equals, loggo.ERROR)
	c.Assert(second.LogLevel(), gc.Equals, loggo.UNSPECIFIED)
	c.Assert(second.EffectiveLogLevel(), gc.Equals, loggo.ERROR)

	first.SetLogLevel(loggo.DEBUG)
	c.Assert(root.LogLevel(), gc.Equals, loggo.ERROR)
	c.Assert(root.EffectiveLogLevel(), gc.Equals, loggo.ERROR)
	c.Assert(first.LogLevel(), gc.Equals, loggo.DEBUG)
	c.Assert(first.EffectiveLogLevel(), gc.Equals, loggo.DEBUG)
	c.Assert(second.LogLevel(), gc.Equals, loggo.UNSPECIFIED)
	c.Assert(second.EffectiveLogLevel(), gc.Equals, loggo.DEBUG)

	second.SetLogLevel(loggo.INFO)
	c.Assert(root.LogLevel(), gc.Equals, loggo.ERROR)
	c.Assert(root.EffectiveLogLevel(), gc.Equals, loggo.ERROR)
	c.Assert(first.LogLevel(), gc.Equals, loggo.DEBUG)
	c.Assert(first.EffectiveLogLevel(), gc.Equals, loggo.DEBUG)
	c.Assert(second.LogLevel(), gc.Equals, loggo.INFO)
	c.Assert(second.EffectiveLogLevel(), gc.Equals, loggo.INFO)

	first.SetLogLevel(loggo.UNSPECIFIED)
	c.Assert(root.LogLevel(), gc.Equals, loggo.ERROR)
	c.Assert(root.EffectiveLogLevel(), gc.Equals, loggo.ERROR)
	c.Assert(first.LogLevel(), gc.Equals, loggo.UNSPECIFIED)
	c.Assert(first.EffectiveLogLevel(), gc.Equals, loggo.ERROR)
	c.Assert(second.LogLevel(), gc.Equals, loggo.INFO)
	c.Assert(second.EffectiveLogLevel(), gc.Equals, loggo.INFO)
}

type GlobalWritersSuite struct {
	logger loggo.Logger
	writer *loggo.TestWriter
}

var _ = gc.Suite(&GlobalWritersSuite{})

func (s *GlobalWritersSuite) SetUpTest(c *gc.C) {
	loggo.ResetLoggers()
}

func (s *GlobalWritersSuite) TearDownTest(c *gc.C) {
	loggo.ResetWriters()
}

func (s *GlobalWritersSuite) TestLoggerUsesDefault(c *gc.C) {
	logger, writer := newTraceLogger("test.writer")

	logger.Infof("message")

	checkLastMessage(c, writer, "message")
}

func (s *GlobalWritersSuite) TestLoggerUsesMultiple(c *gc.C) {
	logger, writer := newTraceLogger("test.writer")
	err := loggo.RegisterWriter("test", writer, loggo.TRACE)
	c.Assert(err, gc.IsNil)

	logger.Infof("message")

	log := writer.Log()
	c.Check(log, gc.HasLen, 2)
	c.Check(log[0].Message, gc.Equals, "message")
	c.Check(log[1].Message, gc.Equals, "message")
}

func (s *GlobalWritersSuite) TestLoggerRespectsWriterLevel(c *gc.C) {
	logger, writer := newTraceLogger("test.writer")
	loggo.RemoveWriter("default")
	err := loggo.RegisterWriter("test", writer, loggo.ERROR)
	c.Assert(err, gc.IsNil)

	logger.Infof("message")

	c.Check(writer.Log(), gc.HasLen, 0)
}

func (*GlobalWritersSuite) TestRemoveDefaultWriter(c *gc.C) {
	defaultWriter, level, err := loggo.RemoveWriter("default")
	c.Assert(err, gc.IsNil)
	c.Assert(level, gc.Equals, loggo.TRACE)
	c.Assert(defaultWriter, gc.NotNil)

	// Trying again fails.
	defaultWriter, level, err = loggo.RemoveWriter("default")
	c.Assert(err, gc.ErrorMatches, `Writer "default" is not registered`)
	c.Assert(level, gc.Equals, loggo.UNSPECIFIED)
	c.Assert(defaultWriter, gc.IsNil)
}

func (*GlobalWritersSuite) TestRegisterWriterExistingName(c *gc.C) {
	err := loggo.RegisterWriter("default", &loggo.TestWriter{}, loggo.INFO)
	c.Assert(err, gc.ErrorMatches, `there is already a Writer registered with the name "default"`)
}

func (*GlobalWritersSuite) TestRegisterNilWriter(c *gc.C) {
	err := loggo.RegisterWriter("nil", nil, loggo.INFO)
	c.Assert(err, gc.ErrorMatches, `Writer cannot be nil`)
}

func (*GlobalWritersSuite) TestRegisterWriterTypedNil(c *gc.C) {
	// If the interface is a typed nil, we have to trust the user.
	var writer *loggo.TestWriter
	err := loggo.RegisterWriter("nil", writer, loggo.INFO)
	c.Assert(err, gc.IsNil)
}

func (*GlobalWritersSuite) TestReplaceDefaultWriter(c *gc.C) {
	oldWriter, err := loggo.ReplaceDefaultWriter(&loggo.TestWriter{})
	c.Assert(oldWriter, gc.NotNil)
	c.Assert(err, gc.IsNil)
}

func (*GlobalWritersSuite) TestReplaceDefaultWriterWithNil(c *gc.C) {
	oldWriter, err := loggo.ReplaceDefaultWriter(nil)
	c.Assert(oldWriter, gc.IsNil)
	c.Assert(err, gc.ErrorMatches, "Writer cannot be nil")
}

func (*GlobalWritersSuite) TestReplaceDefaultWriterNoDefault(c *gc.C) {
	loggo.RemoveWriter("default")
	oldWriter, err := loggo.ReplaceDefaultWriter(&loggo.TestWriter{})
	c.Assert(oldWriter, gc.IsNil)
	c.Assert(err, gc.ErrorMatches, `there is no "default" writer`)
}

func (s *GlobalWritersSuite) TestWillWrite(c *gc.C) {
	// By default, the root logger watches TRACE messages
	c.Assert(loggo.WillWrite(loggo.TRACE), gc.Equals, true)
	// Note: ReplaceDefaultWriter doesn't let us change the default log
	//	 level :(
	writer, _, err := loggo.RemoveWriter("default")
	c.Assert(err, gc.IsNil)
	c.Assert(writer, gc.NotNil)
	err = loggo.RegisterWriter("default", writer, loggo.CRITICAL)
	c.Assert(err, gc.IsNil)
	c.Assert(loggo.WillWrite(loggo.TRACE), gc.Equals, false)
	c.Assert(loggo.WillWrite(loggo.DEBUG), gc.Equals, false)
	c.Assert(loggo.WillWrite(loggo.INFO), gc.Equals, false)
	c.Assert(loggo.WillWrite(loggo.WARNING), gc.Equals, false)
	c.Assert(loggo.WillWrite(loggo.CRITICAL), gc.Equals, true)
}

type GlobalBenchmarksSuite struct {
	logger loggo.Logger
	writer *loggo.TestWriter
}

var _ = gc.Suite(&GlobalBenchmarksSuite{})

func (s *GlobalBenchmarksSuite) SetUpTest(c *gc.C) {
	loggo.ResetLoggers()
	s.logger, s.writer = newTraceLogger("test.writer")
	loggo.RemoveWriter("default")
	err := loggo.RegisterWriter("test", s.writer, loggo.TRACE)
	c.Assert(err, gc.IsNil)
}

func (s *GlobalBenchmarksSuite) TearDownTest(c *gc.C) {
	loggo.ResetWriters()
}

func (s *GlobalBenchmarksSuite) BenchmarkLoggingNoWriters(c *gc.C) {
	// No writers
	loggo.RemoveWriter("test")
	for i := 0; i < c.N; i++ {
		s.logger.Warningf("just a simple warning for %d", i)
	}
}

func (s *GlobalBenchmarksSuite) BenchmarkLoggingNoWritersNoFormat(c *gc.C) {
	// No writers
	loggo.RemoveWriter("test")
	for i := 0; i < c.N; i++ {
		s.logger.Warningf("just a simple warning")
	}
}

func (s *GlobalBenchmarksSuite) BenchmarkLoggingTestWriters(c *gc.C) {
	for i := 0; i < c.N; i++ {
		s.logger.Warningf("just a simple warning for %d", i)
	}
	c.Assert(s.writer.Log, gc.HasLen, c.N)
}

func (s *GlobalBenchmarksSuite) BenchmarkLoggingDiskWriter(c *gc.C) {
	logFile, cleanup := setupTempFileWriter(c)
	defer cleanup()
	msg := "just a simple warning for %d"
	for i := 0; i < c.N; i++ {
		s.logger.Warningf(msg, i)
	}
	offset, err := logFile.Seek(0, os.SEEK_CUR)
	c.Assert(err, gc.IsNil)
	c.Assert((offset > int64(len(msg))*int64(c.N)), gc.Equals, true,
		gc.Commentf("Not enough data was written to the log file."))
}

func (s *GlobalBenchmarksSuite) BenchmarkLoggingDiskWriterNoMessages(c *gc.C) {
	logFile, cleanup := setupTempFileWriter(c)
	defer cleanup()
	// Change the log level
	writer, _, err := loggo.RemoveWriter("testfile")
	c.Assert(err, gc.IsNil)
	loggo.RegisterWriter("testfile", writer, loggo.WARNING)
	msg := "just a simple warning for %d"
	for i := 0; i < c.N; i++ {
		s.logger.Debugf(msg, i)
	}
	offset, err := logFile.Seek(0, os.SEEK_CUR)
	c.Assert(err, gc.IsNil)
	c.Assert(offset, gc.Equals, int64(0),
		gc.Commentf("Data was written to the log file."))
}

func (s *GlobalBenchmarksSuite) BenchmarkLoggingDiskWriterNoMessagesLogLevel(c *gc.C) {
	logFile, cleanup := setupTempFileWriter(c)
	defer cleanup()
	// Change the log level
	s.logger.SetLogLevel(loggo.WARNING)
	msg := "just a simple warning for %d"
	for i := 0; i < c.N; i++ {
		s.logger.Debugf(msg, i)
	}
	offset, err := logFile.Seek(0, os.SEEK_CUR)
	c.Assert(err, gc.IsNil)
	c.Assert(offset, gc.Equals, int64(0),
		gc.Commentf("Data was written to the log file."))
}

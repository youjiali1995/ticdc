//// Copyright 2020 PingCAP, Inc.
////
//// Licensed under the Apache License, Version 2.0 (the "License");
//// you may not use this file except in compliance with the License.
//// You may obtain a copy of the License at
////
////     http://www.apache.org/licenses/LICENSE-2.0
////
//// Unless required by applicable law or agreed to in writing, software
//// distributed under the License is distributed on an "AS IS" BASIS,
//// See the License for the specific language governing permissions and
//// limitations under the License.
//
package cdc

//
//import (
//	"context"
//	"fmt"
//	"io"
//	"sync"
//	"sync/atomic"
//	"time"
//
//	"github.com/cenkalti/backoff"
//	"github.com/google/uuid"
//	"github.com/pingcap/errors"
//	"github.com/pingcap/failpoint"
//	"github.com/pingcap/log"
//	"github.com/pingcap/ticdc/cdc/entry"
//	"github.com/pingcap/ticdc/cdc/kv"
//	"github.com/pingcap/ticdc/cdc/model"
//	"github.com/pingcap/ticdc/cdc/puller"
//	"github.com/pingcap/ticdc/cdc/sink"
//	cerror "github.com/pingcap/ticdc/pkg/errors"
//	"github.com/pingcap/ticdc/pkg/filter"
//	"github.com/pingcap/ticdc/pkg/notify"
//	tablepipeline "github.com/pingcap/ticdc/pkg/processor/pipeline"
//	"github.com/pingcap/ticdc/pkg/regionspan"
//	"github.com/pingcap/ticdc/pkg/security"
//	"github.com/pingcap/ticdc/pkg/util"
//	pd "github.com/tikv/pd/client"
//	"go.etcd.io/etcd/clientv3/concurrency"
//	"go.uber.org/zap"
//	"golang.org/x/sync/errgroup"
//)
//
//const (
//	// defaultMemBufferCapacity is the default memory buffer per change feed.
//	defaultMemBufferCapacity int64 = 10 * 1024 * 1024 * 1024 // 10G
//
//	defaultSyncResolvedBatch = 1024
//
//	schemaStorageGCLag = time.Minute * 20
//)
//
//type aprocessor struct {
//	id           string
//	captureInfo  model.CaptureInfo
//	changefeedID string
//	changefeed   model.ChangeFeedInfo
//	limitter     *puller.BlurResourceLimitter
//	stopped      int32
//
//	pdCli      pd.Client
//	credential *security.Credential
//	etcdCli    kv.CDCEtcdClient
//	session    *concurrency.Session
//
//	sinkManager *sink.Manager
//
//	globalResolvedTs        uint64
//	localResolvedTs         uint64
//	checkpointTs            uint64
//	globalcheckpointTs      uint64
//	flushCheckpointInterval time.Duration
//
//	ddlPuller       puller.Puller
//	ddlPullerCancel context.CancelFunc
//	schemaStorage   *entry.SchemaStorage
//
//<<<<<<< HEAD
//	outputFromTable chan *model.PolymorphicEvent
//	output2Sink     chan *model.PolymorphicEvent
//	mounter         entry.Mounter
//=======
//	mounter entry.Mounter
//>>>>>>> pingcap/master
//
//	stateMu           sync.Mutex
//	status            *model.TaskStatus
//	position          *model.TaskPosition
//	tables            map[int64]*tablepipeline.TablePipeline
//	markTableIDs      map[int64]struct{}
//	statusModRevision int64
//
//<<<<<<< HEAD
//=======
//	globalResolvedTsNotifier  *notify.Notifier
//>>>>>>> pingcap/master
//	localResolvedNotifier     *notify.Notifier
//	localResolvedReceiver     *notify.Receiver
//	localCheckpointTsNotifier *notify.Notifier
//	localCheckpointTsReceiver *notify.Receiver
//
//<<<<<<< HEAD
//	wg    *errgroup.Group
//	errCh chan<- error
//=======
//	wg       *errgroup.Group
//	errCh    chan<- error
//	opDoneCh chan int64
//}
//
//type tableInfo struct {
//	id           int64
//	name         string // quoted schema and table, used in metircs only
//	resolvedTs   uint64
//	checkpointTs uint64
//
//	markTableID   int64
//	mResolvedTs   uint64
//	mCheckpointTs uint64
//	workload      model.WorkloadInfo
//	cancel        context.CancelFunc
//}
//
//func (t *tableInfo) loadResolvedTs() uint64 {
//	tableRts := atomic.LoadUint64(&t.resolvedTs)
//	if t.markTableID != 0 {
//		mTableRts := atomic.LoadUint64(&t.mResolvedTs)
//		if mTableRts < tableRts {
//			return mTableRts
//		}
//	}
//	return tableRts
//}
//
//func (t *tableInfo) loadCheckpointTs() uint64 {
//	tableCkpt := atomic.LoadUint64(&t.checkpointTs)
//	if t.markTableID != 0 {
//		mTableCkpt := atomic.LoadUint64(&t.mCheckpointTs)
//		if mTableCkpt < tableCkpt {
//			return mTableCkpt
//		}
//	}
//	return tableCkpt
//>>>>>>> pingcap/master
//}
//
//// newProcessor creates and returns a processor for the specified change feed
//func newProcessor(
//	ctx context.Context,
//	pdCli pd.Client,
//	credential *security.Credential,
//	session *concurrency.Session,
//	changefeed model.ChangeFeedInfo,
//	sinkManager *sink.Manager,
//	changefeedID string,
//	captureInfo model.CaptureInfo,
//	checkpointTs uint64,
//	errCh chan error,
//	flushCheckpointInterval time.Duration,
//) (*processor, error) {
//	etcdCli := session.Client()
//	cdcEtcdCli := kv.NewCDCEtcdClient(ctx, etcdCli)
//	limitter := puller.NewBlurResourceLimmter(defaultMemBufferCapacity)
//
//	log.Info("start processor with startts",
//		zap.Uint64("startts", checkpointTs), util.ZapFieldChangefeed(ctx))
//	kvStorage, err := util.KVStorageFromCtx(ctx)
//	if err != nil {
//		return nil, errors.Trace(err)
//	}
//	ddlspans := []regionspan.Span{regionspan.GetDDLSpan(), regionspan.GetAddIndexDDLSpan()}
//	ddlPuller := puller.NewPuller(ctx, pdCli, credential, kvStorage, checkpointTs, ddlspans, limitter, false)
//	filter, err := filter.NewFilter(changefeed.Config)
//	if err != nil {
//		return nil, errors.Trace(err)
//	}
//	schemaStorage, err := createSchemaStorage(kvStorage, checkpointTs, filter, changefeed.Config.ForceReplicate)
//	if err != nil {
//		return nil, errors.Trace(err)
//	}
//
//	localResolvedNotifier := new(notify.Notifier)
//	localCheckpointTsNotifier := new(notify.Notifier)
//<<<<<<< HEAD
//=======
//	globalResolvedTsNotifier := new(notify.Notifier)
//>>>>>>> pingcap/master
//	localResolvedReceiver, err := localResolvedNotifier.NewReceiver(50 * time.Millisecond)
//	if err != nil {
//		return nil, err
//	}
//	localCheckpointTsReceiver, err := localCheckpointTsNotifier.NewReceiver(50 * time.Millisecond)
//	if err != nil {
//		return nil, err
//	}
//
//	p := &processor{
//		id:            uuid.New().String(),
//		limitter:      limitter,
//		captureInfo:   captureInfo,
//		changefeedID:  changefeedID,
//		changefeed:    changefeed,
//		pdCli:         pdCli,
//		credential:    credential,
//		etcdCli:       cdcEtcdCli,
//		session:       session,
//		sinkManager:   sinkManager,
//		ddlPuller:     ddlPuller,
//		schemaStorage: schemaStorage,
//		errCh:         errCh,
//
//		flushCheckpointInterval: flushCheckpointInterval,
//
//<<<<<<< HEAD
//		position:        &model.TaskPosition{CheckPointTs: checkpointTs},
//		outputFromTable: make(chan *model.PolymorphicEvent),
//		output2Sink:     make(chan *model.PolymorphicEvent, defaultOutputChanSize),
//=======
//		position: &model.TaskPosition{CheckPointTs: checkpointTs},
//>>>>>>> pingcap/master
//
//		globalResolvedTsNotifier: globalResolvedTsNotifier,
//		localResolvedNotifier:    localResolvedNotifier,
//		localResolvedReceiver:    localResolvedReceiver,
//
//		checkpointTs:              checkpointTs,
//		localCheckpointTsNotifier: localCheckpointTsNotifier,
//		localCheckpointTsReceiver: localCheckpointTsReceiver,
//
//		tables:       make(map[int64]*tablepipeline.TablePipeline),
//		markTableIDs: make(map[int64]struct{}),
//	}
//	modRevision, status, err := p.etcdCli.GetTaskStatus(ctx, p.changefeedID, p.captureInfo.ID)
//	if err != nil {
//		return nil, errors.Trace(err)
//	}
//	p.status = status
//	p.statusModRevision = modRevision
//
//	for tableID, replicaInfo := range p.status.Tables {
//		p.addTable(ctx, tableID, replicaInfo)
//	}
//	return p, nil
//}
//
//func (p *processor) Run(ctx context.Context) {
//	wg, cctx := errgroup.WithContext(ctx)
//	p.wg = wg
//	ddlPullerCtx, ddlPullerCancel :=
//		context.WithCancel(util.PutTableInfoInCtx(cctx, 0, "ticdc-processor-ddl"))
//	p.ddlPullerCancel = ddlPullerCancel
//
//	wg.Go(func() error {
//		return p.positionWorker(cctx)
//	})
//
//	wg.Go(func() error {
//		return p.globalStatusWorker(cctx)
//	})
//
//	wg.Go(func() error {
//<<<<<<< HEAD
//		return p.syncResolved(cctx)
//	})
//
//	wg.Go(func() error {
//		return p.collectMetrics(cctx)
//	})
//
//	wg.Go(func() error {
//=======
//>>>>>>> pingcap/master
//		return p.ddlPuller.Run(ddlPullerCtx)
//	})
//
//	wg.Go(func() error {
//		return p.ddlPullWorker(cctx)
//	})
//
//	wg.Go(func() error {
//		return p.mounter.Run(cctx)
//	})
//
//	wg.Go(func() error {
//		return p.workloadWorker(cctx)
//	})
//
//	go func() {
//		if err := wg.Wait(); err != nil {
//			select {
//			case p.errCh <- err:
//			default:
//			}
//		}
//	}()
//}
//
//// wait blocks until all routines in processor are returned
//func (p *processor) wait() {
//	err := p.wg.Wait()
//	if err != nil && errors.Cause(err) != context.Canceled {
//		log.Error("processor wait error",
//			zap.String("capture-id", p.captureInfo.ID),
//			zap.String("capture", p.captureInfo.AdvertiseAddr),
//			zap.String("changefeed", p.changefeedID),
//			zap.Error(err),
//		)
//	}
//}
//
//func (p *processor) writeDebugInfo(w io.Writer) {
//	fmt.Fprintf(w, "changefeedID: %s, info: %+v, status: %+v\n", p.changefeedID, p.changefeed, p.status)
//
//	p.stateMu.Lock()
//	for tableID, table := range p.tables {
//		fmt.Fprintf(w, "\ttable id: %d, resolveTS: %d\n", tableID, table.ResolvedTs())
//	}
//	p.stateMu.Unlock()
//
//	fmt.Fprintf(w, "\n")
//}
//
//<<<<<<< HEAD
//=======
//// localResolvedWorker do the flowing works.
//// 1, update resolve ts by scanning all table's resolve ts.
//// 2, update checkpoint ts by consuming entry from p.executedTxns.
//// 3, sync TaskStatus between in memory and storage.
//// 4, check admin command in TaskStatus and apply corresponding command
//func (p *processor) positionWorker(ctx context.Context) error {
//	lastFlushTime := time.Now()
//	retryFlushTaskStatusAndPosition := func() error {
//		t0Update := time.Now()
//		err := retry.Run(500*time.Millisecond, 3, func() error {
//			inErr := p.flushTaskStatusAndPosition(ctx)
//			if inErr != nil {
//				if errors.Cause(inErr) != context.Canceled {
//					logError := log.Error
//					errField := zap.Error(inErr)
//					if cerror.ErrAdminStopProcessor.Equal(inErr) {
//						logError = log.Warn
//						errField = zap.String("error", inErr.Error())
//					}
//					logError("update info failed", util.ZapFieldChangefeed(ctx), errField)
//				}
//				if p.isStopped() || cerror.ErrAdminStopProcessor.Equal(inErr) {
//					return backoff.Permanent(cerror.ErrAdminStopProcessor.FastGenByArgs())
//				}
//			}
//			return inErr
//		})
//		updateInfoDuration.
//			WithLabelValues(p.captureInfo.AdvertiseAddr).
//			Observe(time.Since(t0Update).Seconds())
//		if err != nil {
//			return errors.Annotate(err, "failed to update info")
//		}
//		return nil
//	}
//
//	defer func() {
//		p.localResolvedReceiver.Stop()
//		p.localCheckpointTsReceiver.Stop()
//
//		if !p.isStopped() {
//			err := retryFlushTaskStatusAndPosition()
//			if err != nil && errors.Cause(err) != context.Canceled {
//				log.Warn("failed to update info before exit", util.ZapFieldChangefeed(ctx), zap.Error(err))
//			}
//		}
//
//		log.Info("Local resolved worker exited", util.ZapFieldChangefeed(ctx))
//	}()
//
//	resolvedTsGauge := resolvedTsGauge.WithLabelValues(p.changefeedID, p.captureInfo.AdvertiseAddr)
//	metricResolvedTsLagGauge := resolvedTsLagGauge.WithLabelValues(p.changefeedID, p.captureInfo.AdvertiseAddr)
//	checkpointTsGauge := checkpointTsGauge.WithLabelValues(p.changefeedID, p.captureInfo.AdvertiseAddr)
//	metricCheckpointTsLagGauge := checkpointTsLagGauge.WithLabelValues(p.changefeedID, p.captureInfo.AdvertiseAddr)
//	for {
//		select {
//		case <-ctx.Done():
//			return ctx.Err()
//		case <-p.localResolvedReceiver.C:
//			minResolvedTs := p.ddlPuller.GetResolvedTs()
//			p.stateMu.Lock()
//			for _, table := range p.tables {
//				ts := table.loadResolvedTs()
//
//				if ts < minResolvedTs {
//					minResolvedTs = ts
//				}
//			}
//			p.stateMu.Unlock()
//			atomic.StoreUint64(&p.localResolvedTs, minResolvedTs)
//
//			phyTs := oracle.ExtractPhysical(minResolvedTs)
//			// It is more accurate to get tso from PD, but in most cases we have
//			// deployed NTP service, a little bias is acceptable here.
//			metricResolvedTsLagGauge.Set(float64(oracle.GetPhysical(time.Now())-phyTs) / 1e3)
//			resolvedTsGauge.Set(float64(phyTs))
//
//			if p.position.ResolvedTs < minResolvedTs {
//				p.position.ResolvedTs = minResolvedTs
//				if err := retryFlushTaskStatusAndPosition(); err != nil {
//					return errors.Trace(err)
//				}
//			}
//		case <-p.localCheckpointTsReceiver.C:
//			checkpointTs := atomic.LoadUint64(&p.globalResolvedTs)
//			p.stateMu.Lock()
//			for _, table := range p.tables {
//				ts := table.loadCheckpointTs()
//				if ts < checkpointTs {
//					checkpointTs = ts
//				}
//			}
//			p.stateMu.Unlock()
//			if checkpointTs == 0 {
//				log.Warn("0 is not a valid checkpointTs", util.ZapFieldChangefeed(ctx))
//				continue
//			}
//			atomic.StoreUint64(&p.checkpointTs, checkpointTs)
//			phyTs := oracle.ExtractPhysical(checkpointTs)
//			// It is more accurate to get tso from PD, but in most cases we have
//			// deployed NTP service, a little bias is acceptable here.
//			metricCheckpointTsLagGauge.Set(float64(oracle.GetPhysical(time.Now())-phyTs) / 1e3)
//
//			if time.Since(lastFlushTime) < p.flushCheckpointInterval {
//				continue
//			}
//
//			p.position.CheckPointTs = checkpointTs
//			checkpointTsGauge.Set(float64(phyTs))
//			if err := retryFlushTaskStatusAndPosition(); err != nil {
//				return errors.Trace(err)
//			}
//			lastFlushTime = time.Now()
//		}
//	}
//}
//
//func (p *processor) ddlPullWorker(ctx context.Context) error {
//	ddlRawKVCh := puller.SortOutput(ctx, p.ddlPuller.Output())
//	var ddlRawKV *model.RawKVEntry
//	for {
//		select {
//		case <-ctx.Done():
//			return errors.Trace(ctx.Err())
//		case ddlRawKV = <-ddlRawKVCh:
//		}
//		if ddlRawKV == nil {
//			continue
//		}
//		failpoint.Inject("processorDDLResolved", func() {})
//		if ddlRawKV.OpType == model.OpTypeResolved {
//			p.schemaStorage.AdvanceResolvedTs(ddlRawKV.CRTs)
//			p.localResolvedNotifier.Notify()
//		}
//		job, err := entry.UnmarshalDDL(ddlRawKV)
//		if err != nil {
//			return errors.Trace(err)
//		}
//		if job == nil {
//			continue
//		}
//		if err := p.schemaStorage.HandleDDLJob(job); err != nil {
//			return errors.Trace(err)
//		}
//	}
//}
//
//func (p *processor) workloadWorker(ctx context.Context) error {
//	t := time.NewTicker(10 * time.Second)
//	err := p.etcdCli.PutTaskWorkload(ctx, p.changefeedID, p.captureInfo.ID, nil)
//	if err != nil {
//		return errors.Trace(err)
//	}
//	for {
//		select {
//		case <-ctx.Done():
//			return errors.Trace(ctx.Err())
//		case <-t.C:
//		}
//		if p.isStopped() {
//			continue
//		}
//		p.stateMu.Lock()
//		workload := make(model.TaskWorkload, len(p.tables))
//		for _, table := range p.tables {
//			workload[table.id] = table.workload
//		}
//		p.stateMu.Unlock()
//		err := p.etcdCli.PutTaskWorkload(ctx, p.changefeedID, p.captureInfo.ID, &workload)
//		if err != nil {
//			return errors.Trace(err)
//		}
//	}
//}
//
//>>>>>>> pingcap/master
//func (p *processor) flushTaskPosition(ctx context.Context) error {
//	failpoint.Inject("ProcessorUpdatePositionDelaying", func() {
//		time.Sleep(1 * time.Second)
//	})
//	if p.isStopped() {
//		return cerror.ErrAdminStopProcessor.GenWithStackByArgs()
//	}
//	// p.position.Count = p.sink.Count()
//	updated, err := p.etcdCli.PutTaskPositionOnChange(ctx, p.changefeedID, p.captureInfo.ID, p.position)
//	if err != nil {
//		if errors.Cause(err) != context.Canceled {
//			log.Error("failed to flush task position", util.ZapFieldChangefeed(ctx), zap.Error(err))
//			return errors.Trace(err)
//		}
//	}
//	if updated {
//		log.Debug("flushed task position", util.ZapFieldChangefeed(ctx), zap.Stringer("position", p.position))
//	}
//	return nil
//}
//
//// First try to synchronize task status from etcd.
//// If local cached task status is outdated (caused by new table scheduling),
//// update it to latest value, and force update task position, since add new
//// tables may cause checkpoint ts fallback in processor.
//func (p *processor) flushTaskStatusAndPosition(ctx context.Context) error {
//	if p.isStopped() {
//		return cerror.ErrAdminStopProcessor.GenWithStackByArgs()
//	}
//	var tablesToRemove []model.TableID
//	newTaskStatus, newModRevision, err := p.etcdCli.AtomicPutTaskStatus(ctx, p.changefeedID, p.captureInfo.ID,
//		func(modRevision int64, taskStatus *model.TaskStatus) (bool, error) {
//			// if the task status is not changed and not operation to handle
//			// we need not to change the task status
//			if p.statusModRevision == modRevision && !taskStatus.SomeOperationsUnapplied() {
//				return false, nil
//			}
//			// task will be stopped in capture task handler, do nothing
//			if taskStatus.AdminJobType.IsStopState() {
//				return false, backoff.Permanent(cerror.ErrAdminStopProcessor.GenWithStackByArgs())
//			}
//			toRemove, err := p.handleTables(ctx, taskStatus)
//			tablesToRemove = append(tablesToRemove, toRemove...)
//			if err != nil {
//				return false, backoff.Permanent(errors.Trace(err))
//			}
//			// processor reads latest task status from etcd, analyzes operation
//			// field and processes table add or delete. If operation is unapplied
//			// but stays unchanged after processor handling tables, it means no
//			// status is changed and we don't need to flush task status neigher.
//			if !taskStatus.Dirty {
//				return false, nil
//			}
//			err = p.flushTaskPosition(ctx)
//			return true, err
//		})
//	if err != nil {
//		// not need to check error
//		//nolint:errcheck
//		p.flushTaskPosition(ctx)
//		return errors.Trace(err)
//	}
//	for _, tableID := range tablesToRemove {
//		p.removeTable(tableID)
//	}
//	// newModRevision == 0 means status is not updated
//	if newModRevision > 0 {
//		p.statusModRevision = newModRevision
//		p.status = newTaskStatus
//	}
//	syncTableNumGauge.
//		WithLabelValues(p.changefeedID, p.captureInfo.AdvertiseAddr).
//		Set(float64(len(p.status.Tables)))
//
//	return p.flushTaskPosition(ctx)
//}
//
//func (p *processor) removeTable(tableID int64) {
//	p.stateMu.Lock()
//	defer p.stateMu.Unlock()
//
//	log.Debug("remove table", zap.String("changefeed", p.changefeedID), zap.Int64("id", tableID))
//
//	table, ok := p.tables[tableID]
//	if !ok {
//		log.Warn("table not found", zap.String("changefeed", p.changefeedID), zap.Int64("tableID", tableID))
//		return
//	}
//
//<<<<<<< HEAD
//	if table.Status() != tablepipeline.TableStatusStopped {
//		return
//	}
//	table.Cancel()
//=======
//	table.cancel()
//>>>>>>> pingcap/master
//	delete(p.tables, tableID)
//	_, markTableID := table.ID()
//	if markTableID != 0 {
//		delete(p.markTableIDs, markTableID)
//	}
//	tableResolvedTsGauge.DeleteLabelValues(p.changefeedID, p.captureInfo.AdvertiseAddr, table.Name())
//	syncTableNumGauge.WithLabelValues(p.changefeedID, p.captureInfo.AdvertiseAddr).Dec()
//}
//
//<<<<<<< HEAD
//// syncResolved handle `p.ddlJobsCh` and `p.resolvedTxns`
//func (p *processor) syncResolved(ctx context.Context) error {
//	defer func() {
//		log.Info("syncResolved stopped", util.ZapFieldChangefeed(ctx))
//	}()
//=======
//// handleTables handles table scheduler on this processor, add or remove table puller
//func (p *processor) handleTables(ctx context.Context, status *model.TaskStatus) (tablesToRemove []model.TableID, err error) {
//	for tableID, opt := range status.Operation {
//		if opt.TableProcessed() {
//			continue
//		}
//		if opt.Delete {
//			if opt.BoundaryTs <= p.position.CheckPointTs {
//				if opt.BoundaryTs != p.position.CheckPointTs {
//					log.Warn("the replication progresses beyond the BoundaryTs and duplicate data may be received by downstream",
//						zap.Uint64("local resolved TS", p.position.ResolvedTs), zap.Any("opt", opt))
//				}
//				table, exist := p.tables[tableID]
//				if !exist {
//					log.Warn("table which will be deleted is not found",
//						util.ZapFieldChangefeed(ctx), zap.Int64("tableID", tableID))
//					opt.Done = true
//					opt.Status = model.OperFinished
//					status.Dirty = true
//					continue
//				}
//				table.cancel()
//				checkpointTs := table.loadCheckpointTs()
//				log.Debug("stop table", zap.Int64("tableID", tableID),
//					util.ZapFieldChangefeed(ctx),
//					zap.Any("opt", opt),
//					zap.Uint64("checkpointTs", checkpointTs))
//				opt.BoundaryTs = checkpointTs
//				tablesToRemove = append(tablesToRemove, tableID)
//				opt.Done = true
//				opt.Status = model.OperFinished
//				status.Dirty = true
//			}
//		} else {
//			replicaInfo, exist := status.Tables[tableID]
//			if !exist {
//				return tablesToRemove, cerror.ErrProcessorTableNotFound.GenWithStack("replicaInfo of table(%d)", tableID)
//			}
//			if p.changefeed.Config.Cyclic.IsEnabled() && replicaInfo.MarkTableID == 0 {
//				return tablesToRemove, cerror.ErrProcessorTableNotFound.GenWithStack("normal table(%d) and mark table not match ", tableID)
//			}
//			p.addTable(ctx, tableID, replicaInfo)
//			opt.Status = model.OperProcessed
//			status.Dirty = true
//		}
//	}
//
//	for {
//		select {
//		case <-ctx.Done():
//			return nil, ctx.Err()
//		case tableID := <-p.opDoneCh:
//			log.Debug("Operation done signal received",
//				util.ZapFieldChangefeed(ctx),
//				zap.Int64("tableID", tableID),
//				zap.Reflect("operation", status.Operation[tableID]))
//			if status.Operation[tableID] == nil {
//				log.Debug("TableID does not exist, probably a mark table, ignore",
//					util.ZapFieldChangefeed(ctx), zap.Int64("tableID", tableID))
//				continue
//			}
//			status.Operation[tableID].Done = true
//			status.Operation[tableID].Status = model.OperFinished
//			status.Dirty = true
//		default:
//			goto done
//		}
//	}
//done:
//	if !status.SomeOperationsUnapplied() {
//		status.Operation = nil
//		// status.Dirty must be true when status changes from `unapplied` to `applied`,
//		// setting status.Dirty = true is not **must** here.
//		status.Dirty = true
//	}
//	return tablesToRemove, nil
//}
//
//// globalStatusWorker read global resolve ts from changefeed level info and forward `tableInputChans` regularly.
//func (p *processor) globalStatusWorker(ctx context.Context) error {
//	log.Info("Global status worker started", util.ZapFieldChangefeed(ctx))
//
//	var (
//		changefeedStatus *model.ChangeFeedStatus
//		statusRev        int64
//		lastCheckPointTs uint64
//		lastResolvedTs   uint64
//		watchKey         = kv.GetEtcdKeyJob(p.changefeedID)
//	)
//
//	updateStatus := func(changefeedStatus *model.ChangeFeedStatus) {
//		atomic.StoreUint64(&p.globalcheckpointTs, changefeedStatus.CheckpointTs)
//		if lastResolvedTs == changefeedStatus.ResolvedTs &&
//			lastCheckPointTs == changefeedStatus.CheckpointTs {
//			return
//		}
//		if lastResolvedTs < changefeedStatus.ResolvedTs {
//			lastResolvedTs = changefeedStatus.ResolvedTs
//			atomic.StoreUint64(&p.globalResolvedTs, lastResolvedTs)
//			log.Debug("Update globalResolvedTs",
//				zap.Uint64("globalResolvedTs", lastResolvedTs), util.ZapFieldChangefeed(ctx))
//			p.globalResolvedTsNotifier.Notify()
//		}
//	}
//
//	retryCfg := backoff.WithMaxRetries(
//		backoff.WithContext(
//			backoff.NewExponentialBackOff(), ctx),
//		5,
//	)
//	for {
//		select {
//		case <-ctx.Done():
//			log.Info("Global resolved worker exited", util.ZapFieldChangefeed(ctx))
//			return ctx.Err()
//		default:
//		}
//
//		err := backoff.Retry(func() error {
//			var err error
//			changefeedStatus, statusRev, err = p.etcdCli.GetChangeFeedStatus(ctx, p.changefeedID)
//			if err != nil {
//				if errors.Cause(err) == context.Canceled {
//					return backoff.Permanent(err)
//				}
//				log.Error("Global resolved worker: read global resolved ts failed",
//					util.ZapFieldChangefeed(ctx), zap.Error(err))
//			}
//			return err
//		}, retryCfg)
//		if err != nil {
//			return errors.Trace(err)
//		}
//
//		updateStatus(changefeedStatus)
//
//		ch := p.etcdCli.Client.Watch(ctx, watchKey, clientv3.WithRev(statusRev+1), clientv3.WithFilterDelete())
//		for resp := range ch {
//			if resp.Err() == mvcc.ErrCompacted {
//				break
//			}
//			if resp.Err() != nil {
//				return cerror.WrapError(cerror.ErrProcessorEtcdWatch, err)
//			}
//			for _, ev := range resp.Events {
//				var status model.ChangeFeedStatus
//				if err := status.Unmarshal(ev.Kv.Value); err != nil {
//					return err
//				}
//				updateStatus(&status)
//			}
//		}
//	}
//}
//
//func createSchemaStorage(
//	kvStorage tidbkv.Storage,
//	checkpointTs uint64,
//	filter *filter.Filter,
//	forceReplicate bool,
//) (*entry.SchemaStorage, error) {
//	meta, err := kv.GetSnapshotMeta(kvStorage, checkpointTs)
//	if err != nil {
//		return nil, errors.Trace(err)
//	}
//	return entry.NewSchemaStorage(meta, checkpointTs, filter, forceReplicate)
//}
//
//func (p *processor) addTable(ctx context.Context, tableID int64, replicaInfo *model.TableReplicaInfo) {
//	p.stateMu.Lock()
//	defer p.stateMu.Unlock()
//
//	var tableName string
//	err := retry.Run(time.Millisecond*5, 3, func() error {
//		if name, ok := p.schemaStorage.GetLastSnapshot().GetTableNameByID(tableID); ok {
//			tableName = name.QuoteString()
//			return nil
//		}
//		return errors.Errorf("failed to get table name, fallback to use table id: %d", tableID)
//	})
//	if err != nil {
//		log.Warn("get table name for metric", util.ZapFieldChangefeed(ctx), zap.String("error", err.Error()))
//		tableName = strconv.Itoa(int(tableID))
//	}
//
//	if _, ok := p.tables[tableID]; ok {
//		log.Warn("Ignore existing table", util.ZapFieldChangefeed(ctx), zap.Int64("ID", tableID))
//		return
//	}
//
//	globalcheckpointTs := atomic.LoadUint64(&p.globalcheckpointTs)
//
//	if replicaInfo.StartTs < globalcheckpointTs {
//		log.Warn("addTable: startTs < checkpoint",
//			util.ZapFieldChangefeed(ctx),
//			zap.Int64("tableID", tableID),
//			zap.Uint64("checkpoint", globalcheckpointTs),
//			zap.Uint64("startTs", replicaInfo.StartTs))
//	}
//
//	globalResolvedTs := atomic.LoadUint64(&p.globalResolvedTs)
//	log.Debug("Add table", zap.Int64("tableID", tableID),
//		util.ZapFieldChangefeed(ctx),
//		zap.String("name", tableName),
//		zap.Any("replicaInfo", replicaInfo),
//		zap.Uint64("globalResolvedTs", globalResolvedTs))
//
//	ctx = util.PutTableInfoInCtx(ctx, tableID, tableName)
//	ctx, cancel := context.WithCancel(ctx)
//	table := &tableInfo{
//		id:         tableID,
//		name:       tableName,
//		resolvedTs: replicaInfo.StartTs,
//	}
//	// TODO(leoppro) calculate the workload of this table
//	// We temporarily set the value to constant 1
//	table.workload = model.WorkloadInfo{Workload: 1}
//
//	startPuller := func(tableID model.TableID, pResolvedTs *uint64, pCheckpointTs *uint64) sink.Sink {
//		// start table puller
//		enableOldValue := p.changefeed.Config.EnableOldValue
//		span := regionspan.GetTableSpan(tableID, enableOldValue)
//		kvStorage, err := util.KVStorageFromCtx(ctx)
//		if err != nil {
//			p.errCh <- err
//			return nil
//		}
//		plr := puller.NewPuller(ctx, p.pdCli, p.credential, kvStorage, replicaInfo.StartTs, []regionspan.Span{span}, p.limitter, enableOldValue)
//		go func() {
//			err := plr.Run(ctx)
//			if errors.Cause(err) != context.Canceled {
//				p.errCh <- err
//			}
//		}()
//
//		var sorter puller.EventSorter
//		switch p.changefeed.Engine {
//		case model.SortInMemory:
//			sorter = puller.NewEntrySorter()
//		case model.SortInFile, model.SortUnified:
//			err := util.IsDirAndWritable(p.changefeed.SortDir)
//			if err != nil {
//				if os.IsNotExist(errors.Cause(err)) {
//					err = os.MkdirAll(p.changefeed.SortDir, 0o755)
//					if err != nil {
//						p.errCh <- errors.Annotate(cerror.WrapError(cerror.ErrProcessorSortDir, err), "create dir")
//						return nil
//					}
//				} else {
//					p.errCh <- errors.Annotate(cerror.WrapError(cerror.ErrProcessorSortDir, err), "sort dir check")
//					return nil
//				}
//			}
//
//			if p.changefeed.Engine == model.SortInFile {
//				sorter = puller.NewFileSorter(p.changefeed.SortDir)
//			} else {
//				// Unified Sorter
//				sorter = psorter.NewUnifiedSorter(p.changefeed.SortDir, tableName, util.CaptureAddrFromCtx(ctx))
//			}
//		default:
//			p.errCh <- cerror.ErrUnknownSortEngine.GenWithStackByArgs(p.changefeed.Engine)
//			return nil
//		}
//		go func() {
//			err := sorter.Run(ctx)
//			if errors.Cause(err) != context.Canceled {
//				p.errCh <- err
//			}
//		}()
//
//		go func() {
//			p.pullerConsume(ctx, plr, sorter)
//		}()
//
//		tableSink := p.sinkManager.CreateTableSink(tableID, replicaInfo.StartTs)
//		go func() {
//			p.sorterConsume(ctx, tableID, tableName, sorter, pResolvedTs, pCheckpointTs, replicaInfo, tableSink)
//		}()
//		return tableSink
//	}
//	var tableSink, mTableSink sink.Sink
//	if p.changefeed.Config.Cyclic.IsEnabled() && replicaInfo.MarkTableID != 0 {
//		mTableID := replicaInfo.MarkTableID
//		// we should to make sure a mark table is only listened once.
//		if _, exist := p.markTableIDs[mTableID]; !exist {
//			p.markTableIDs[mTableID] = struct{}{}
//			table.markTableID = mTableID
//			table.mResolvedTs = replicaInfo.StartTs
//
//			mTableSink = startPuller(mTableID, &table.mResolvedTs, &table.mCheckpointTs)
//		}
//	}
//
//	p.tables[tableID] = table
//	if p.position.CheckPointTs > replicaInfo.StartTs {
//		p.position.CheckPointTs = replicaInfo.StartTs
//	}
//	if p.position.ResolvedTs > replicaInfo.StartTs {
//		p.position.ResolvedTs = replicaInfo.StartTs
//	}
//
//	atomic.StoreUint64(&p.localResolvedTs, p.position.ResolvedTs)
//	tableSink = startPuller(tableID, &table.resolvedTs, &table.checkpointTs)
//	table.cancel = func() {
//		cancel()
//		tableSink.Close()
//		if mTableSink != nil {
//			mTableSink.Close()
//		}
//	}
//	syncTableNumGauge.WithLabelValues(p.changefeedID, p.captureInfo.AdvertiseAddr).Inc()
//}
//
//// sorterConsume receives sorted PolymorphicEvent from sorter of each table and
//// sends to processor's output chan
//func (p *processor) sorterConsume(
//	ctx context.Context,
//	tableID int64,
//	tableName string,
//	sorter puller.EventSorter,
//	pResolvedTs *uint64,
//	pCheckpointTs *uint64,
//	replicaInfo *model.TableReplicaInfo,
//	sink sink.Sink,
//) {
//	var lastResolvedTs uint64
//	opDone := false
//	resolvedTsGauge := tableResolvedTsGauge.WithLabelValues(p.changefeedID, p.captureInfo.AdvertiseAddr, tableName)
//	checkDoneTicker := time.NewTicker(1 * time.Second)
//	checkDone := func() {
//		localResolvedTs := atomic.LoadUint64(&p.localResolvedTs)
//		globalResolvedTs := atomic.LoadUint64(&p.globalResolvedTs)
//		if !opDone && lastResolvedTs >= localResolvedTs && localResolvedTs >= globalResolvedTs {
//			log.Debug("localResolvedTs >= globalResolvedTs, sending operation done signal",
//				zap.Uint64("localResolvedTs", localResolvedTs), zap.Uint64("globalResolvedTs", globalResolvedTs),
//				zap.Int64("tableID", tableID), util.ZapFieldChangefeed(ctx))
//
//			opDone = true
//			checkDoneTicker.Stop()
//			select {
//			case <-ctx.Done():
//				if errors.Cause(ctx.Err()) != context.Canceled {
//					p.errCh <- ctx.Err()
//				}
//				return
//			case p.opDoneCh <- tableID:
//			}
//		}
//		if !opDone {
//			log.Debug("addTable not done",
//				util.ZapFieldChangefeed(ctx),
//				zap.Uint64("tableResolvedTs", lastResolvedTs),
//				zap.Uint64("localResolvedTs", localResolvedTs),
//				zap.Uint64("globalResolvedTs", globalResolvedTs),
//				zap.Int64("tableID", tableID))
//		}
//	}
//>>>>>>> pingcap/master
//
//	events := make([]*model.PolymorphicEvent, 0, defaultSyncResolvedBatch)
//	rows := make([]*model.RowChangedEvent, 0, defaultSyncResolvedBatch)
//
//<<<<<<< HEAD
//	flush2Sink := func() error {
//=======
//	flushRowChangedEvents := func() error {
//>>>>>>> pingcap/master
//		for _, ev := range events {
//			err := ev.WaitPrepare(ctx)
//			if err != nil {
//				return errors.Trace(err)
//			}
//			if ev.Row == nil {
//				continue
//			}
//			rows = append(rows, ev.Row)
//		}
//		failpoint.Inject("ProcessorSyncResolvedPreEmit", func() {
//			log.Info("Prepare to panic for ProcessorSyncResolvedPreEmit")
//			time.Sleep(10 * time.Second)
//			panic("ProcessorSyncResolvedPreEmit")
//		})
//<<<<<<< HEAD
//		if len(rows) == 0 {
//			return nil
//		}
//		for _, row := range rows {
//			log.Info("LEOPPRO: show row", zap.Reflect("row", row))
//		}
//		err := p.sink.EmitRowChangedEvents(ctx, rows...)
//=======
//		err := sink.EmitRowChangedEvents(ctx, rows...)
//>>>>>>> pingcap/master
//		if err != nil {
//			return errors.Trace(err)
//		}
//		events = events[:0]
//		rows = rows[:0]
//		return nil
//	}
//
//	processRowChangedEvent := func(row *model.PolymorphicEvent) error {
//		events = append(events, row)
//
//		if len(events) >= defaultSyncResolvedBatch {
//<<<<<<< HEAD
//			err := flush2Sink()
//=======
//			err := flushRowChangedEvents()
//>>>>>>> pingcap/master
//			if err != nil {
//				return errors.Trace(err)
//			}
//		}
//		return nil
//	}
//
//<<<<<<< HEAD
//	metricFlushDuration := sinkFlushRowChangedDuration.WithLabelValues(p.changefeedID, p.captureInfo.AdvertiseAddr)
//
//	flushSink := func(resolvedTs model.Ts) error {
//		globalResolvedTs := atomic.LoadUint64(&p.globalResolvedTs)
//		if resolvedTs > globalResolvedTs {
//			resolvedTs = globalResolvedTs
//		}
//		if resolvedTs == 0 || atomic.LoadUint64(&p.checkpointTs) == resolvedTs {
//			return nil
//		}
//		start := time.Now()
//
//		checkpointTs, err := p.sink.FlushRowChangedEvents(ctx, resolvedTs)
//		if err != nil {
//			return errors.Trace(err)
//		}
//		if checkpointTs != 0 {
//			atomic.StoreUint64(&p.checkpointTs, checkpointTs)
//			p.localCheckpointTsNotifier.Notify()
//		}
//
//		dur := time.Since(start)
//		metricFlushDuration.Observe(dur.Seconds())
//		if dur > 3*time.Second {
//			log.Warn("flush row changed events too slow",
//				zap.Duration("duration", dur), util.ZapFieldChangefeed(ctx))
//		}
//		return nil
//	}
//
//	var resolvedTs uint64
//	for {
//		select {
//		case <-ctx.Done():
//			return ctx.Err()
//		case row := <-p.output2Sink:
//			if row == nil {
//				continue
//			}
//			failpoint.Inject("ProcessorSyncResolvedError", func() {
//				failpoint.Return(errors.New("processor sync resolved injected error"))
//			})
//			if row.RawKV != nil && row.RawKV.OpType == model.OpTypeResolved {
//				resolvedTs = row.CRTs
//				if err := flush2Sink(); err != nil {
//					return errors.Trace(err)
//				}
//				if err := flushSink(resolvedTs); err != nil {
//					return errors.Trace(err)
//				}
//				continue
//			}
//			// Global resolved ts should fallback in some table rebalance cases,
//			// since the start-ts(from checkpoint ts) or a rebalanced table could
//			// be less then the global resolved ts.
//			localResolvedTs := atomic.LoadUint64(&p.localResolvedTs)
//			if resolvedTs > localResolvedTs {
//				log.Info("global resolved ts fallback",
//					zap.String("changefeed", p.changefeedID),
//					zap.Uint64("localResolvedTs", localResolvedTs),
//					zap.Uint64("resolvedTs", resolvedTs),
//				)
//				resolvedTs = localResolvedTs
//			}
//			if row.CRTs <= resolvedTs {
//				_ = row.WaitPrepare(ctx)
//				log.Panic("The CRTs must be greater than the resolvedTs",
//					zap.String("model", "processor"),
//					zap.String("changefeed", p.changefeedID),
//					zap.Uint64("resolvedTs", resolvedTs),
//					zap.Any("row", row))
//			}
//			err := processRowChangedEvent(row)
//			if err != nil {
//				return errors.Trace(err)
//=======
//	globalResolvedTsReceiver, err := p.globalResolvedTsNotifier.NewReceiver(1 * time.Second)
//	if err != nil {
//		if errors.Cause(err) != context.Canceled {
//			p.errCh <- errors.Trace(err)
//		}
//		return
//	}
//	defer globalResolvedTsReceiver.Stop()
//
//	for {
//		select {
//		case <-ctx.Done():
//			if errors.Cause(ctx.Err()) != context.Canceled {
//				p.errCh <- ctx.Err()
//			}
//			return
//		case pEvent := <-sorter.Output():
//			if pEvent == nil {
//				continue
//			}
//
//			pEvent.SetUpFinishedChan()
//			select {
//			case <-ctx.Done():
//				if errors.Cause(ctx.Err()) != context.Canceled {
//					p.errCh <- ctx.Err()
//				}
//				return
//			case p.mounter.Input() <- pEvent:
//			}
//
//			if pEvent.RawKV != nil && pEvent.RawKV.OpType == model.OpTypeResolved {
//				if pEvent.CRTs == 0 {
//					continue
//				}
//				err := flushRowChangedEvents()
//				if err != nil {
//					if errors.Cause(err) != context.Canceled {
//						p.errCh <- errors.Trace(err)
//					}
//					return
//				}
//				atomic.StoreUint64(pResolvedTs, pEvent.CRTs)
//				lastResolvedTs = pEvent.CRTs
//				p.localResolvedNotifier.Notify()
//				resolvedTsGauge.Set(float64(oracle.ExtractPhysical(pEvent.CRTs)))
//				if !opDone {
//					checkDone()
//				}
//				continue
//			}
//			if pEvent.CRTs <= lastResolvedTs || pEvent.CRTs < replicaInfo.StartTs {
//				log.Panic("The CRTs of event is not expected, please report a bug",
//					util.ZapFieldChangefeed(ctx),
//					zap.String("model", "sorter"),
//					zap.Uint64("resolvedTs", lastResolvedTs),
//					zap.Int64("tableID", tableID),
//					zap.Any("replicaInfo", replicaInfo),
//					zap.Any("row", pEvent))
//			}
//			failpoint.Inject("ProcessorSyncResolvedError", func() {
//				p.errCh <- errors.New("processor sync resolved injected error")
//				failpoint.Return()
//			})
//			err := processRowChangedEvent(pEvent)
//			if err != nil {
//				if errors.Cause(err) != context.Canceled {
//					p.errCh <- errors.Trace(err)
//				}
//				return
//			}
//		case <-globalResolvedTsReceiver.C:
//			localResolvedTs := atomic.LoadUint64(&p.localResolvedTs)
//			globalResolvedTs := atomic.LoadUint64(&p.globalResolvedTs)
//			var minTs uint64
//			if localResolvedTs < globalResolvedTs {
//				minTs = localResolvedTs
//				log.Warn("the local resolved ts is less than the global resolved ts",
//					zap.Uint64("localResolvedTs", localResolvedTs), zap.Uint64("globalResolvedTs", globalResolvedTs))
//			} else {
//				minTs = globalResolvedTs
//			}
//			if minTs == 0 || atomic.LoadUint64(&p.checkpointTs) == minTs {
//				continue
//			}
//
//			checkpointTs, err := sink.FlushRowChangedEvents(ctx, minTs)
//			if err != nil {
//				if errors.Cause(err) != context.Canceled {
//					p.errCh <- errors.Trace(err)
//				}
//				return
//			}
//			if checkpointTs != 0 {
//				atomic.StoreUint64(pCheckpointTs, checkpointTs)
//				p.localCheckpointTsNotifier.Notify()
//			}
//		case <-checkDoneTicker.C:
//			if !opDone {
//				checkDone()
//>>>>>>> pingcap/master
//			}
//		}
//	}
//}
//

//
//func (p *processor) isStopped() bool {
//	return atomic.LoadInt32(&p.stopped) == 1
//}
//
//var runProcessorImpl = runProcessor

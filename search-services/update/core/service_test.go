package core_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"yadro.com/course/update/core"
)

var testLog = slog.New(slog.NewTextHandler(os.Stderr, nil))

type mockDB struct {
	ids        []int
	stats      core.DBStats
	addErr     error
	idsErr     error
	dropCalled bool
}

// Возвращает список ID комиксов из поля ids, или ошибку из idsErr
func (m *mockDB) IDs(_ context.Context) ([]int, error) { return m.ids, m.idsErr }

// Сохраняет комикс; возвращает addErr если он задан
func (m *mockDB) Add(_ context.Context, _ core.Comics) error { return m.addErr }

// Возвращает статистику из поля stats
func (m *mockDB) Stats(_ context.Context) (core.DBStats, error) { return m.stats, nil }

// Помечает dropCalled=true чтобы тест мог проверить факт вызова
func (m *mockDB) Drop(_ context.Context) error {
	m.dropCalled = true
	return nil
}

type mockXKCD struct {
	lastID    int
	lastIDErr error
	comics    map[int]core.XKCDInfo
	getErr    error
}

// Возвращает lastID или lastIDErr
func (m *mockXKCD) LastID(_ context.Context) (int, error) { return m.lastID, m.lastIDErr }

// Ищет комикс по id в карте comics; если getErr задан — возвращает его
func (m *mockXKCD) Get(_ context.Context, id int) (core.XKCDInfo, error) {
	if m.getErr != nil {
		return core.XKCDInfo{}, m.getErr
	}
	c, ok := m.comics[id]
	if !ok {
		return core.XKCDInfo{}, errors.New("not found")
	}
	return c, nil
}

type mockWords struct{}

// Возвращает ["keyword"] для любой непустой фразы
func (m *mockWords) Norm(_ context.Context, phrase string) ([]string, error) {
	if phrase == "" {
		return nil, nil
	}
	return []string{"keyword"}, nil
}

type mockPublisher struct {
	updated int
	dropped int
}

// Считает сколько раз было опубликовано событие обновления
func (m *mockPublisher) PublishUpdated() { m.updated++ }

// Считает сколько раз было опубликовано событие удаления
func (m *mockPublisher) PublishDropped() { m.dropped++ }

// Нулевая конкурентность — ошибка при создании сервиса
func TestNewService_BadConcurrency(t *testing.T) {
	_, err := core.NewService(testLog, &mockDB{}, &mockXKCD{}, &mockWords{}, &mockPublisher{}, 0)
	require.Error(t, err)
}

// Корректные параметры — сервис создаётся без ошибок
func TestNewService_OK(t *testing.T) {
	svc, err := core.NewService(testLog, &mockDB{}, &mockXKCD{}, &mockWords{}, &mockPublisher{}, 1)
	require.NoError(t, err)
	require.NotNil(t, svc)
}

// Сразу после создания статус должен быть idle
func TestStatus_IdleByDefault(t *testing.T) {
	svc, _ := core.NewService(testLog, &mockDB{}, &mockXKCD{}, &mockWords{}, &mockPublisher{}, 1)
	require.Equal(t, core.StatusIdle, svc.Status(context.Background()))
}

// Если все комиксы уже есть в БД — ничего не скачиваем и не публикуем
func TestUpdate_NothingToDo(t *testing.T) {
	db := &mockDB{ids: []int{1, 2, 3}}
	xkcd := &mockXKCD{lastID: 3}
	pub := &mockPublisher{}
	svc, _ := core.NewService(testLog, db, xkcd, &mockWords{}, pub, 2)
	require.NoError(t, svc.Update(context.Background()))
	require.Equal(t, 0, pub.updated)
}

// Новые комиксы скачиваются и публикуется событие обновления
func TestUpdate_FetchesAndPublishes(t *testing.T) {
	db := &mockDB{ids: []int{}}
	xkcd := &mockXKCD{
		lastID: 2,
		comics: map[int]core.XKCDInfo{
			1: {ID: 1, URL: "url1", Description: "desc one"},
			2: {ID: 2, URL: "url2", Description: "desc two"},
		},
	}
	pub := &mockPublisher{}
	svc, _ := core.NewService(testLog, db, xkcd, &mockWords{}, pub, 2)
	require.NoError(t, svc.Update(context.Background()))
	require.Equal(t, 1, pub.updated)
}

// Второй вызов Update пока первый ещё выполняется — ErrUpdateRunning
func TestUpdate_ConcurrentCallReturnsError(t *testing.T) {
	ready := make(chan struct{})
	unblock := make(chan struct{})

	blockingXKCD := &blockXKCD{ready: ready, unblock: unblock, inner: &mockXKCD{lastID: 0}}
	pub := &mockPublisher{}
	svc, _ := core.NewService(testLog, &mockDB{ids: []int{}}, blockingXKCD, &mockWords{}, pub, 1)

	done := make(chan error, 1)
	go func() { done <- svc.Update(context.Background()) }()

	<-ready

	err := svc.Update(context.Background())
	require.ErrorIs(t, err, core.ErrUpdateRunning)

	close(unblock)
	<-done
}

// drop вызывает DB.Drop и публикует событие
func TestDrop_CallsPublisher(t *testing.T) {
	db := &mockDB{}
	pub := &mockPublisher{}
	svc, _ := core.NewService(testLog, db, &mockXKCD{}, &mockWords{}, pub, 1)
	require.NoError(t, svc.Drop(context.Background()))
	require.True(t, db.dropCalled)
	require.Equal(t, 1, pub.dropped)
}

// ComicsTotal = lastID когда lastID < 404
func TestStats_ReturnsCorrectTotal(t *testing.T) {
	db := &mockDB{stats: core.DBStats{ComicsFetched: 5}}
	xkcd := &mockXKCD{lastID: 10}
	svc, _ := core.NewService(testLog, db, xkcd, &mockWords{}, &mockPublisher{}, 1)
	stats, err := svc.Stats(context.Background())
	require.NoError(t, err)
	require.Equal(t, 10, stats.ComicsTotal)
}

// Комикс #404 не существует — вычитаем единицу из lastID
func TestStats_SubtractsMissing404(t *testing.T) {
	db := &mockDB{stats: core.DBStats{ComicsFetched: 5}}
	xkcd := &mockXKCD{lastID: 500}
	svc, _ := core.NewService(testLog, db, xkcd, &mockWords{}, &mockPublisher{}, 1)
	stats, err := svc.Stats(context.Background())
	require.NoError(t, err)
	require.Equal(t, 499, stats.ComicsTotal)
}

// blockXKCD блокирует LastID чтобы симулировать долгое обновление
type blockXKCD struct {
	ready   chan struct{}
	unblock chan struct{}
	inner   *mockXKCD
	first   bool
}

// При первом вызове сигналит в ready и блокируется до unblock — имитирует долгий запрос
func (b *blockXKCD) LastID(ctx context.Context) (int, error) {
	if !b.first {
		b.first = true
		close(b.ready)
		<-b.unblock
	}
	return b.inner.LastID(ctx)
}

// Делегирует вызов внутреннему моку
func (b *blockXKCD) Get(ctx context.Context, id int) (core.XKCDInfo, error) {
	return b.inner.Get(ctx, id)
}
